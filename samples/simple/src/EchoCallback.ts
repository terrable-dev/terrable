import { DoPromise } from "./Utils";

const handler = (event, context, callback) => {
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