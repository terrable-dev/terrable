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
            console.log('Error executing node process code.', error);
            context.complete();
        } finally {
            context = createContext();
        }
    }
});

function createContext() {
    const consoleProxy = {
        log: console.log,
        error: console.error,
        warn: console.warn,
        info: console.info,
    };
        
    return vm.createContext({
        ...global,
        console: consoleProxy,
        require: require,
        process: process,
        complete: () => {
            consoleProxy.log("CODE_EXECUTION_COMPLETE");
            buffer = "";
            process.stdin.resume();
        },
    })
}
