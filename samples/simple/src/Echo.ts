import { MyUtil } from "./Utils";

const handler = async (event) => {
    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            hello: 'world',
            number: MyUtil(),
        }),
    }
}

export { handler };