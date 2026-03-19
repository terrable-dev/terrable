const handler = async (event) => {
    const [record] = event.Records;

    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            recordCount: event.Records.length,
            firstRecord: {
                body: record.body,
                eventSource: record.eventSource,
                eventSourceARN: record.eventSourceARN,
                awsRegion: record.awsRegion,
                approximateReceiveCount: record.attributes.ApproximateReceiveCount,
            },
        }),
    }
}

export { handler };
