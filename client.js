const net		= require("net");
const readline	= require("readline");

const client = net.createConnection({
	host: process.argv[2] || "localhost",
	port: 6060
})

client.on("connect", () => {
	process.stdout.write("\x1B[38;2;32;208;32mINFO: Connected\x1B[0m\n");

	const rl = readline.createInterface({
		input: process.stdin,
	});

	client.on("data", data => {
		console.log("\x1B[38;2;100;150;255m>", data.toString(), "\x1B[0m");
	})

	rl.on("line", async(line) => {
		client.write(line);
	})
})
