# 刑事トラック＝漫GO
*"...Georgia Tech University."*

## What is this?
decatholac MANGO (dM) is a Discord bot that fetches new manga chapter releases and then announce it to servers it's been registered to.

Currently it can parse from HTML, JSON and RSS.

## Building
- ```go test``` to make sure it runs fine.
- Copy ```config.sample.toml``` into ```config.toml``` and make changes.
- ```go run .``` or ```go build``` to build and/or run it.

## Commands
- ```/set-as-feed-channel``` to set the current channel as the feed channel.
- ```/fetch``` to trigger the bot to fetch for new chapters from the source.
- ```/announce``` to trigger the bot to announce new chapters to the feed channel.

Fetching and announcing happens periodically through a cronjob.
The two commands listed above can be used to trigger it manually.

## Source configuration
It's kind of a pain to explain how it works so just look at ```config.sample.toml```
and the ```(parser)_test.go``` files and find out how it works.

It's pretty simple anyway.