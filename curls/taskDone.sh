curl -v 'http://localhost:8080/task-update' \
  -H 'Content-Type: application/json' \
  -d '{
	"floorId": "669007c9276d50f367b2187e",
	"task": {
		"id": "0",
		"assignedTo": 0
	},
	"action": "DONE"
  }'
