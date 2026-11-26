package main

import "errors"

var ErrNoEmptyURLs = errors.New("no empty URLS")
var ErrURLTooLong = errors.New("URL too long")
var ErrProblemNormalizingURL = errors.New("problem normalizing URL")

var ErrSchemeMustBeHTTPOrHTTPS = errors.New("scheme must be http or https")
var ErrURLCouldNotBeParsed = errors.New("URL could not be parsed")
var ErrMissingHost = errors.New("missing host")

var ErrNoFreeCode = errors.New("couldnt find a free code")
