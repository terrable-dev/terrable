const vm = require('vm');

process.stdin.setEncoding('utf8');

let context = createContext();

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
        } finally {
            context = createContext();
        }
    }
});

function createContext() {
    return vm.createContext({
        ...global,
        console: console,
        require: require,
        process: process,
        complete: () => {
            console.log("CODE_EXECUTION_COMPLETE");
            process.stdin.resume();
        },
    })
}