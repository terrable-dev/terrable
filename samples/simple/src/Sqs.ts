const handler = async (event) => {
    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            queryStringParameters: event.queryStringParameters,
            event: event,
            env: process.env,
        }),
    }
}

export { handler };