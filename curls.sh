curl -v 'http://192.168.1.9:8080/floor/' \
--header 'Content-Type: application/json' \
--data '{
  "floorName": "Floor1A",
  "residents": [
    "762b569bffebb4b815cd5e78",
    "762b5ace2337d3c989bcc238",
    "762b5f46cd8a580b287a8d84"
  ],
  "tasks": [
    {
			"id": "1",
      "name": "Gelbersack entfernen",
      "assignedTo": "662b5f46cd8a580b287a8d84"
    },
    {
			"id": "2",
      "name": "Biomüll wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    },
    {
			"id": "3",
      "name": "Restmull wegbringen",
      "assignedTo": "662b569bffebb4b815cd5e78"
    }
  ],
  "rooms": [
    {
			"id": "1",
      "number": "1",
			"order": 1,
      "resident": "762b569bffebb4b815cd5e78"
    },
    {
			"id": "2",
      "number": "2",
			"order": 2,
      "resident": "762b5ace2337d3c989bcc238"
    },
    {
			"id": "3",
      "number": "3",
			"order": 3,
      "resident": "762b5f46cd8a580b287a8d84"
    }
  ]
}'
