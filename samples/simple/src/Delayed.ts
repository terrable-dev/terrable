import { DoPromise } from "./Utils";

const handler = async (event) => {
    const before = new Date().getTime();

    await DoPromise(3000);

    const after = new Date().getTime();

    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            time: after - before,
        }),
    }
}

export { handler };