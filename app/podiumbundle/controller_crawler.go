package podiumbundle

import (
	"net/http"
	"strings"
	"log"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"errors"
)

func (c *PodiumController) CrawlPodiatristsHandler(w http.ResponseWriter, r *http.Request) {



	pods := make(map[int64]Podiatrist)

	for i := 1; i < 80; i++ {
		log.Println(i)
		url := "https://www.podsfixfeet.co.uk/places"
		if i > 1 {
			url = fmt.Sprintf("https://www.podsfixfeet.co.uk/places/page/%d/", i)
		}
		html := getWebsite(url)

		urls := getPodiatrists(html)
		err, pod := getData(getWebsite(urls[0]))
		if err == nil {
			pods[pod.CrawlId] = *pod
		}

		err, pod = getData(getWebsite(urls[1]))
		if err == nil {
			pods[pod.CrawlId] = *pod
		}
		err, pod = getData(getWebsite(urls[2]))
		if err == nil {
			pods[pod.CrawlId] = *pod
		}
	}
	log.Println(pods)
	log.Println(len(pods))

	for _, pod := range pods {
		c.ormDB.Create(&pod)
	}


}


func getPodiatrists(html string) []string  {
	urls := []string{"", "", ""}
	urls[0], html = getPodUrl(html)
	urls[1], html = getPodUrl(html)
	urls[2], html = getPodUrl(html)
	log.Println(urls)
	return urls

}

func getPodUrl(html string) (string, string) {

	pos := strings.Index(html, `<div class="geodir-content ">`)
	keyLength := len(`<div class="geodir-content ">`)
	html = html[pos+keyLength:]

	pos = strings.Index(html, `<a href="`)
	keyLength = len(`<a href="`)
	pos += keyLength
	posEnd := strings.Index(html, `" clas`)

	return html[pos:posEnd], html
}



func getData(html string) (error, *Podiatrist) {
	pos := strings.Index(html, `<style></style><script type="application/ld+json">`)
	keyLength := len(`<style></style><script type="application/ld+json">`)
	pos += keyLength
	posEnd := strings.Index(html, `</script><meta property="og:image" content="`)

	//log.Println(html)
	log.Println(len(html))
	log.Println(pos)
	log.Println(posEnd)
	if posEnd < pos {
		posEnd = strings.Index(html, `</script><link rel="icon" hr`)
	}

	if posEnd < pos {
		return errors.New("No Pod"), nil
	}

	mapJson := strings.TrimSpace(html[pos:posEnd])
	mapJson = mapJson[0:len(mapJson)]

	log.Println(mapJson)

	crawlerPod := CrawlerPodiatrist{}
	err := json.Unmarshal([]byte(mapJson), &crawlerPod)
	if err != nil {
		log.Println(err)
	}

	log.Println(crawlerPod)

	// id holen
	postId := CrawlerPodiatristPost{}
	pos = strings.Index(html, `var fcaPcPost =`)
	keyLength = len(`var fcaPcPost = `)
	pos += keyLength
	tmpHtml := html[pos:]
	posEnd = strings.Index(tmpHtml, `;`)
	err = json.Unmarshal([]byte(tmpHtml[0:posEnd]), &postId)
	if err != nil {
		log.Println(err)
	}
	log.Println(postId.Id)

	// und nun noch Email ;)
	pos = strings.Index(html,`<input type="hidden" name="recipient-email" value="`)
	keyLength = len(`<input type="hidden" name="recipient-email" value="`)
	pos += keyLength
	tmpHtml = html[pos:]
	posEnd = strings.Index(tmpHtml, `" `)
	email := tmpHtml[0:posEnd]


	// Home visit
	posEnd = strings.Index(html, `"></i> Home Visit</li>`)
	tmp := html[posEnd-5:posEnd]
	hasHomeVisit := false
	if tmp == "check" {
		hasHomeVisit = true
	}

	posEnd = strings.Index(html, `"></i> Clinic</li>`)
	tmp = html[posEnd-5:posEnd]
	hasClinic := false
	if tmp == "check" {
		hasClinic = true
	}

	/*
		// Image holen
		if hasImage {
			pos = posEnd
			keyLength = len(`</script><meta property="og:image" content="`)
			pos += keyLength
			posEnd = strings.Index(html, `"/><link rel="icon" hr`)

			imageLink := strings.TrimSpace(html[pos:posEnd])
			log.Println(imageLink)
		}
		// Home Visit, Clinic
	*/

	website := crawlerPod.Url
	if len(crawlerPod.SameAs) > 0  {
		website = crawlerPod.SameAs[0]
	}
	hasPodium := false
	if strings.Index(html, `Podium-Supplier`) > 0 {
		hasPodium = true
	}

	pod := &Podiatrist {
		Name: crawlerPod.Name,
		Description: crawlerPod.Description,
		Telephone: crawlerPod.Telephone,
		Url: website,
		Image: crawlerPod.Image,
		Review: crawlerPod.Review,
		Email: email,
		StreetAddress: crawlerPod.Address.StreetAddress,
		AddressLocality: crawlerPod.Address.AddressLocality,
		AddressRegion: crawlerPod.Address.AddressRegion,
		AddressCountry: crawlerPod.Address.AddressCountry,
		PostalCode: crawlerPod.Address.PostalCode,

		HasHomeVisit: hasHomeVisit,
		HasClinic: hasClinic,
		HasPodium: hasPodium,
		Latitude: crawlerPod.Geo.Latitude,
		Longitude: crawlerPod.Geo.Longitude,
		CrawlId: postId.Id,
		CrawlSite: crawlerPod.Url,
	}

	return nil, pod

}


func getGeo(html string) (float64, float64) {
	pos := strings.Index(html, "var gd_listing_map = ")
	keyLength := len("var gd_listing_map = ")
	pos += keyLength

	posEnd := strings.Index(html, "var gd_listing_map_jason_args")
	posEnd -= 1

	log.Println(len(html))
	log.Println(pos)
	log.Println(posEnd)

	mapJson := strings.TrimSpace(html[pos:posEnd])
	mapJson = mapJson[0:len(mapJson)-1]

	log.Println(mapJson)

	gdMap := GDMap{}
	err := json.Unmarshal([]byte(mapJson), &gdMap)
	if err != nil {
		log.Println(err)
	}

	log.Println(gdMap.Latitude)
	log.Println(gdMap.Longitude)

	return gdMap.Latitude, gdMap.Longitude
}

func getWebsite(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	return string(html)
}