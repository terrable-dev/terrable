import { DoPromise, MyUtil } from "./Utils";

const handler = async (event) => {
    console.log('ENV 2', process.env);
    
    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            number: MyUtil(),
            evt: event,
            env: process.env,
        }),
    }
}

export { handler };