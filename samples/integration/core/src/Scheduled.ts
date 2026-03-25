const handler = async (event) => {
  return {
    statusCode: 200,
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      source: event.source,
      detailType: event["detail-type"],
      region: event.region,
      resources: event.resources,
      detail: event.detail,
      time: event.time,
    }),
  };
};

export { handler };
