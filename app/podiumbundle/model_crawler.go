package podiumbundle



type Podiatrist struct {
	//core.Model
	Name			string			`json:"name"`
	Description		string			`json:"description"`
	Telephone		string			`json:"telephone"`
	Url				string			`json:"url"`
	Image			string			`json:"image"`
	Review			string			`json:"review"`
	Email			string			`json:"email"`
	StreetAddress				string			`json:"streetAddress"`
	AddressLocality				string			`json:"addressLocality"`
	AddressRegion				string			`json:"addressRegion"`
	AddressCountry				string			`json:"addressCountry"`
	PostalCode					string			`json:"postalCode"`

	HasHomeVisit	bool			`json:"has_home_visit"`
	HasClinic		bool			`json:"has_clinic"`
	HasPodium		bool			`json:"has_podium"`

	Latitude		float64			`json:"latitude"`
	Longitude		float64			`json:"longitude"`

	CrawlSite		string			`json:"-"`
	CrawlId			int64			`json:"-"`
}
type Podiatrists []Podiatrist

type GDMap struct {
	Latitude		float64			`json:"latitude,string"`
	Longitude		float64			`json:"longitude,string"`
}







type CrawlerPodiatrist struct {
	Name			string						`json:"name"`
	Description		string						`json:"description"`
	Telephone		string						`json:"telephone"`
	Url				string						`json:"url"`
	SameAs			[]string					`json:"sameAs"`
	Image			string						`json:"image"`
	Address			CrawlerPodiatristAddress	`json:"address"`
	Geo				CrawlerPodiatristGeo		`json:"geo"`
	Review			string						`json:"review"`
}

type CrawlerPodiatristAddress struct {
	StreetAddress				string			`json:"streetAddress"`
	AddressLocality				string			`json:"addressLocality"`
	AddressRegion				string			`json:"addressRegion"`
	AddressCountry				string			`json:"addressCountry"`
	PostalCode					string			`json:"postalCode"`
}

type CrawlerPodiatristGeo struct {
	Latitude				float64			`json:"latitude,string"`
	Longitude				float64			`json:"longitude,string"`
}


type CrawlerPodiatristPost struct {
	Id				int64			`json:"id,string"`
}
