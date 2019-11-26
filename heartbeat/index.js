const https = require("https");
const endpoint = process.env.HEARTBEAT_ENDPOINT;

exports.handler = function(event, context, callback) {
  https
    .get(endpoint, { headers: { "User-Agent": "dilbert-feed" } }, res => {
      if (res.statusCode != 200) {
        callback("HTTP error: " + res.statusCode);
      } else {
        callback(null, { endpoint, status: res.statusCode });
      }
    })
    .on("error", e => {
      callback(Error(e));
    });
};
