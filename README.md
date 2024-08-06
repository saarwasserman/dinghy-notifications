# Notifications Service

A service to handle user notifications (currently supporting only emails)

note: uses [mailtrap.io](https://mailtrap.io/)

## Build (bin, docker)

See Makefile's -build- commands


## Deploy (k8s)

See Makefile's -deploy- command

Includes  two deployments:
```
1. notifications-api
2. notifications-email-consumer
```
Note: check the deploy yaml files and set the required secrets and env vars


## Databases

<b>Kafka<b>

used for notifications events.

<b>topic:</b> email-triggered
