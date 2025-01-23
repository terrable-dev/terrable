import { DoPromise } from "./Utils";

// Handlers cannot return promises (async / await) and
// also use the 'callback' - this results in an Internal Server Error 
// from API Gateway.

// This handler is used to test this functionality.

const handler = async (event, context, callback) => {
    DoPromise(2000).then(() => {
        callback(null, {
            statusCode: 200,
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                queryStringParameters: event.queryStringParameters,
                event: event,
                context: context,
            }),
        })
    
    })
}

export { handler };