import { DoPromise } from "./Utils";

// Configured in simple-api.tf to time out after 1 second
// This will cause a timeout error, which can be caught by 
// the hurl tests to verify the timout logic

const handler = async (event) => {
    await DoPromise(2000);

    return {
        statusCode: 200,
    }
}

export { handler };