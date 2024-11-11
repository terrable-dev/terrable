const handler = async (event) => {
    return {
        statusCode: 200,
        body: JSON.stringify({
            collision: "1"
        }),
    }
}

export { handler };