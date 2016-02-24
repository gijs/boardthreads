package main

import "bt/db"

type Account struct {
	Addresses []db.Address `json:"addresses"`
}
