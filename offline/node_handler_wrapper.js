const vm = require('vm');

process.stdin.setEncoding('utf8');

let buffer = '';

process.stdin.on('data', (chunk) => {
    buffer += chunk;
    
    if (true) {
        const code = buffer.trim();
        buffer = '';
        
        try {
            const script = new vm.Script(code);
            const context = vm.createContext(global);
            script.runInContext(context);
        } catch (error) {
            console.error('Error executing code:', error);
            console.log("TERRABLE_RESULT_START:" + JSON.stringify({
                statusCode: 500,
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    message: "Internal server error",
                    errorMessage: error.message,
                    errorType: error.name,
                    stackTrace: error.stack
                })
            }) + ":TERRABLE_RESULT_END");
        } finally {
            console.log("CODE_EXECUTION_COMPLETE");
        }
    }
});

process.stdin.resume();
