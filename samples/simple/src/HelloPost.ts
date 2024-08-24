const handler = async (event) => {
    const parsed = JSON.parse(event.body);

    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            body: parsed,
            qs: event.queryStringParameters
        }),
    }
}

export { handler };