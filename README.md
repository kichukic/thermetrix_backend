
 

 

    Text BoxSave files for Patient 
---------------------------------------------
 

Saves uploaded images associated with a patient and a specific measurement to the server and records the file paths in the database. 

HTTP Request 

POST /api/v1/patientImages 

Headers 

Content-Type: multipart/form-data 

Form Data Parameters 

| Field | Type | Description | | -------------- | ------------- | ----------------------------------------- | | patient_id | string | Unique identifier for the patient. | | measurement_id| string | Unique identifier for the measurement. | | DFA | file | The DFA image files to be uploaded. | |  

Responses 

Success Response 

Code: 200 OK 

Content: Files saved successfully 

Error Responses 

Code: 400 Bad Request 

Content: Error message describing the problem with the form data. 

Code: 500 Internal Server Error 

Content: Error message describing the problem with the server or file handling. 

Functionality 

Parses the multipart form data with a maximum size of 10 MB. 

Retrieves patient_id and measurement_id from the form data. 

Iterates through each file header for image tags. 

Saves each file to the specified directory in the server. (Ie patient_files folder ) 

Records the file paths to the database with the associated tags. 

Optionally updates the IsDFA_Complete status in the database for the patient and measurement if DFA images are uploaded and processed. 

Returns a success message upon successful saving of files and database update. 

Notes 

The uploaded files are saved in a directory named patients_files within the current working directory of the server. 

The IsDFA_Complete status is only updated if at least one DFA image file is successfully processed. 

 
 
 
 
 

    Text BoxGet Patient Files 
-------------------------------------

 

Retrieves files associated with a specific patient from the database and provides a status indicating if a certain data analysis (DFA) is complete. 

HTTP Request 

GET api/v1/scanHistory/{MeasurementID} 

URL Parameters 

MeasurementID: The unique identifier of the patient whose files are being requested. 

Responses 

200 OK: Successfully retrieved the patient files. The response body will contain a JSON object with the data analysis completion status and a list of URLs for the patient files. 

404 Not Found: No files were found for the given patient ID. 

500 Internal Server Error: There was a problem fetching the patient files or encoding the response. 

Response Body 

{ 

  "isDFA_Complete": boolean, 

  "patientId": string, 

  "url": [ 

    string 

  ] 

} 

isDFA_Complete: A boolean indicating whether the data analysis (DFA) is complete for the patient. 

patientId: The patient ID for which the files were requested. 

url: An array of strings containing the URLs to access the patient's files. 

Example Request 

GET /api/v1/scanHistory/12345 

Example Successful Response 

{ 

  "isDFA_Complete": true, 

  "patientId": "12345", 

  "url": [ 

    "http://localhost:4001/api/v1/serve-file/file1.pdf" 

  ] 

} 

NOTE : in src , need to specify the baseurl and the end point in a imageURL variable in controller_podium.go in func getPatientImagesFromDB 

 

 

 

 

 

 

 

 

        Text BoxText BoxServe-File 
--------------------------------------------

Endpoint: api/v1/serve-image/{filepath} 

Description 

Serves an image file to the client from the server's filesystem. 

HTTP Method: GET 

URL Parameters: 

filepath: A string representing the relative path to the image file within the patients_files directory. 

Responses: 

Success Response: 

Code: 200 OK 

Content-Type: {mime-type-of-file} 

Content: The image file's content. 

Error Responses: 

Code: 404 Not Found 

Content-Type: text/plain 

Content: "Not Found" (Occurs when the specified image file does not exist or cannot be opened.) 

Code: 500 Internal Server Error 

Content-Type: text/plain 

Content: "Internal Server Error" (Occurs when there is a server-side error, for example, if the server's current working directory cannot be determined.) 

Example Request: 

GET api/v1/serve-image/DFA_test2_09_2024.pdf 

Host: example.com 

Example Response: 

HTTP/1.1 200 OK 

Content-Type: image/jpeg 

 

{binary image data} 

 

NOTE: in app workflow the whole url for this api already generated on ScanHistory api as its response  
 
 
 
 

            Text Box Send Email with Attachments API 
--------------------------------------------------------------------

POST /sendmail1 

This endpoint allows clients to send an email with attachments. It expects multi-part form data to be submitted with the request. 

Request 

Form Fields: 

to (string) - The recipient's email address. 

subject (string) - The subject line of the email. 

body (string) - The body content of the email. 

patient_username (string) - The username of the patient, used for naming the attachment files. 

attachments (file array) - Files to be attached to the email. Must be included in the request with a key of "attachments". 

Constraints: 

Maximum form size is 10 MB. 

The attachments field is required and should not be empty. 

Response 

Success: 

HTTP Status Code: 200 OK 

Content-Type: text/plain 

Body: Email sent successfully 

Errors: 

If form data cannot be parsed or is too large: 

HTTP Status Code: 400 Bad Request 

Content-Type: text/plain 

Body: Error message 

If no attachments are provided: 

HTTP Status Code: 400 Bad Request 

Content-Type: text/plain 

Body: No attachments provided 

If there are file system errors, such as being unable to create the directory or save files: 

HTTP Status Code: 500 Internal Server Error 

Content-Type: text/plain 

Body: Error message 

If sending the email fails: 

HTTP Status Code: 500 Internal Server Error 

Content-Type: text/plain 

Body: Error message 

Example cURL Request: 

curl -X POST http://yourdomain.com/sendmail1 \ 

     -F "to=example@recipient.com" \ 

     -F "subject=Test Subject" \ 

     -F "body=This is a test email." \ 

     -F "patient_username=johndoe" \ 

     -F "attachments=@/path/to/attachment1.pdf" \ 

     -F "attachments=@/path/to/attachment2.jpg" 

 
 

         POST COVER LETTER API
--------------------------------------------

http://localhost:4001/api/v1/cover-letter

method : post (form-data)

  body
___________
patient_id : "text"
measurement_id : "text"
cover_letter : file




        GET COVER LETTER API
-------------------------------------------
http://localhost:4001/api/v1/get-cover-letter/{measurement_id}

method : get

  response 
______________
{
    "cover_letter_url": "http://localhost:4001/api/v1/download-cover-letter/2024031518532.pdf",
    "is_referred": true,
    "measurement_id": "112"
}

on line 1743 
theres few codes which specifies the base url of the servers , change it according to the server address 
for example  : change localhost to server address

   coverLetterURL := ""
    if coverLetter.CoverLetter != "" {
        coverLetterURL = fmt.Sprintf("http://localhost:4001/api/v1/download-cover-letter/%s", filepath.Base(coverLetter.CoverLetter))
    }




          DOWNLOAD COVERLETTER
----------------------------------------------
/download-cover-letter/{filepath}
http://localhost:4001/api/v1/download-cover-letter/20240315185325.pdf

will download the cover letter



     POST NOTES API
---------------------------

http://localhost:4001/api/v1/save-notes

body : {
    "patient_id": "123",
    "measurement_id": "456",
    "notes": "This is a test note"
}


response :Data saved successfully


will insert data into DB notes table





      GET NOTES API
-----------------------------
http://localhost:4001/api/v1/get-notes/456


response :

[
    {
        "patient_id": "123",
        "measurement_id": "456",
        "notes": "This is a test note",
        "CreatedAt": "0001-01-01T00:00:00Z"
    }
]










           [       IMPORTANT INFO             ]
 -------------------------------------------------------

 in       ---controllerpodium.go---    file theres a function called  (getPatientImagesFromDB) in that function on line  1484 a variable called imageURL holds the server response url we need tochange it according the base url we r currently having related to server

 also keep  **patient_files** , **cover_letters** and **PodiumFiles** folders inside the root folder of the project