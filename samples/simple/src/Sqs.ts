import { DoPromise } from "./Utils";

const handler = async (event) => {
    console.log('SQS Example', event)
    await DoPromise(1500);
}

export { handler };