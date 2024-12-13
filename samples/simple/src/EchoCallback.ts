import { DoPromise } from "./Utils";

const handler = async (event, context, callback) => {
    await DoPromise(1000);
    DoPromise(3000).then(() => {
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