#!/bin/bash

go build -o ontology.exe ../main.go
go build -o nodectl.exe ../nodectl.go
# go build -race -o ont.exe ../main.go
# go build -race -o nodectl.exe ../nodectl.go
