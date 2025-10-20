package core

// EmptyParams is an empty struct that can be used
// to represent no parameters instead of passing
// an empty struct like struct{}{}. We can then
// check using reflect if the parameter is empty
var EmptyParams struct{}

var EmptyResult struct{}
