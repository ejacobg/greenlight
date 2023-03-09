package main

import (
	"flag"
	"log"
	"net/http"
)

// Return a basic webpage. This will make a GET request to /v1/healthcheck and render its contents in a <div> element.
const html = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8">
	</head>
	<body>
		<h1>Simple CORS</h1>
		<div id="output"></div>
		<script>
			document.addEventListener('DOMContentLoaded', function() {
				fetch("http://localhost:4000/v1/healthcheck").then(
					function(response) {
						response.text().then(function(text) {
							document.getElementById("output").innerHTML = text;
						});
					},
					function(err) {
						document.getElementById("output").innerHTML = err;
					}
				);
			});
		</script>
	</body>
</html>`

func main() {
	// Allow the application port to be configured via a CLI flag.
	addr := flag.String("addr", ":9000", "Server address")
	flag.Parse()

	log.Printf("starting server on %s", *addr)

	// The webpage will be returned for all requests.
	err := http.ListenAndServe(*addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	log.Fatal(err)
}
