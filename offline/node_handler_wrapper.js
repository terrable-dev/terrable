const vm = require('vm');

process.stdin.setEncoding('utf8');

const context = vm.createContext({
    ...global,
    console: console,
    complete: () => {
        console.log("CODE_EXECUTION_COMPLETE");
        process.stdin.resume();
    }
});

let buffer = '';

process.stdin.on('data', (chunk) => {
    const code = chunk.trim();
    buffer += chunk;

    if (buffer.includes('\n')) {
        try {
            const script = new vm.Script(code);
            script.runInContext(context);
        } catch (error) {
            console.log('Error executing node process code.', error)
            context.complete();
        }
    }
});
