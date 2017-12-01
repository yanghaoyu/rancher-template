package main

import (
)

func main(){
	var params Params 

	params.init()

	meta := newMetadata(params)
	meta.init()

}