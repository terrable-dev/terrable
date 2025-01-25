import { DoPromise } from "./Utils";

const handler = async () => {
    await DoPromise(300);
}

export { handler };