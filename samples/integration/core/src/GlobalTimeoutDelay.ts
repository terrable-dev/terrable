import { DoPromise } from "./Utils";

// Uses the module-level timeout from the integration fixture.
const handler = async () => {
    await DoPromise(4000);

    return {
        statusCode: 200,
    }
}

export { handler };
