import { DoPromise, MyUtil } from "./Utils";

const handler = async (event) => {
    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            hello: await DoPromise(),
            number: MyUtil(),
            evt: event,
            env: process.env,
        }),
    }
}

export { handler };