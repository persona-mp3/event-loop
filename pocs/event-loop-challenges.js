// src: https://event-loop-visualizer-ruby.vercel.app/
// Theres an issue or is just a misunderstanding or non-deterministic
// or idk anything. The site favors the Promise queue over the nextTicker queue
function complex_1() {
	console.log('Start');

	setTimeout(() => {
		console.log('Timeout 1');
		Promise.resolve().then(() => {
			console.log('Promise inside Timeout');
		});
		process.nextTick(() => {
			console.log('Next Tick inside Timeout');
		});
	}, 0);

	Promise.resolve().then(() => {
		console.log('Promise 1');
		setTimeout(() => {
			console.log('Timeout inside Promise');
		}, 0);
	});

	console.log('End');

}

function complex_2() {
	console.log('Start');

	setTimeout(() => {
		console.log('Timeout 1');
		Promise.resolve().then(() => {
			console.log('Promise inside Timeout');
		});
		process.nextTick(() => {
			console.log('Next Tick inside Timeout');
		});
	}, 0);

	Promise.resolve().then(() => {
		console.log('Promise 1');
		setTimeout(() => {
			console.log('Timeout inside Promise');
		}, 0);
	});

	setImmediate(() => {
		console.log('Immediate 1');

	});

	console.log('End');
}
