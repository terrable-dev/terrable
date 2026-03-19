const handler = async () => {
    return {
        statusCode: 200,
        body: JSON.stringify({
            collision: "2"
        }),
    }
}

export { handler };
