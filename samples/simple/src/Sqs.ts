import { DoPromise } from "./Utils";

const handler = async (event) => {
    console.log('SQS: ', JSON.stringify(event));
    await DoPromise(300);
}

export { handler };