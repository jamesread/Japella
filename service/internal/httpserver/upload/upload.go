package upload

import (
	"net/http"
)

func Handle(w http.ResponseWriter, r *http.Request) {
	handleFileUpload(w, r)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// Handle file upload logic here
	// For example, you can parse the form and save the uploaded file
	if err := r.ParseMultipartForm(10 << 20); err != nil { // Limit to 10MB
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Process the uploaded file (e.g., save it to disk or database)
	// ...

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}
