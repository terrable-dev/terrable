import { DoPromise, MyUtil } from "./Utils";

const handler = async (event) => {
    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            number: MyUtil(),
            queryString: event.queryStringParameters,
            evt: event,
            env: process.env,
        }),
    }
}

export { handler };