import fs from "fs"

// this is funny?
// because you push this to the promises queue
// but them continue to open the file synchronously
async function read_file(src) {
	const fd = fs.openSync(src, "rw")
	let content = fd.read()
	return content // now would async return Promise((resolve, reject) => resolve(content))?
}

//  Now this will for sure go into the IO queue
//  And when successfull, the callback (err, fd)
//  gets evicted from the io queue, 
//  But if you labelled the func as `async` would that then just 
//  get wrapped in a promise or will v8 know what to do?
function read_file_cb(src) {
	let result = ""
	fs.open(src, "rw", (err, fd) => {
		if (err) {
			result = err
		}
		result = fd.read()
	})
	return result
}

import {open as p_open} from "node:fs/promises"

// this goes into the promie queue for sure
async function read_file_async(src) {
	const content = await p_open()
}
