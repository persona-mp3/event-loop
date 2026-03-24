const fs = require("fs")

async function make_request() {
	const response = await fetch("pokemon.url")
	const data = await response.json()
}

function timeout_func() {
	setTimeout(make_request, 3000)
}

function main() {
	console.log("inspecting the ast module")
}


main()
