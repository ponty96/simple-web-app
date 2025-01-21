# Simple Web App

## Folder Structure

* `/api` - swagger folder for our endpoints and webhooks
* `/build` - ci, dockerfiles e.t.c
* `/internal` - every sub packages and stuff will be here. db, redis, rabbitmq, http
* `/cmd` - entrypoint to startup/run the application
  * `app.go` - this will import everything, read config, check for config, start the server e.t.c and return an `App.Run()` function
  * `main.go` - this will simply run `App.Run()` and is the ultimate entrypoint.


## Project
The project is simple:

A Webhook API that receives Orders and sends out notifications.
|> POST /api/webhook/order
|> check Redis Cache to see if the secret key in the HEADER of the webhook is there
|> publish the Order Created event to RabbitMQ

|> consume the Order Created event
|> DB -> create the Order
|> DB -> create the Order Items
|> DB -> create the Invoice
|> run some random func that just simulates work like sending notifications.


A Rest API
-> POST /api/clients add secret key to Redis [client_id|secret_key]
-> fetch orders for client_id
    -> ratelimit requests using redis to 10 reqs/1 mins
