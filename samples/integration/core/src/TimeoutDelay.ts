import { DoPromise } from "./Utils";

// Configured by the core offline integration fixture to time out after 1 second.
const handler = async () => {
    await DoPromise(2000);

    return {
        statusCode: 200,
    }
}

export { handler };
