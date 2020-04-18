import { Context, Callback } from 'aws-lambda';
import { get } from 'https';

export function handler(event: any, _: Context, callback: Callback) {
  console.log('event: ', JSON.stringify(event));

  const endpoint = process.env.HEARTBEAT_ENDPOINT;
  if (!endpoint) {
    throw new Error('heartbeat endpoint must be set');
  }
  const headers = { 'User-Agent': 'dilbert-feed' };

  get(endpoint, { headers }, (res) => {
    if (res.statusCode != 200) {
      callback('HTTP error: ' + res.statusCode);
    } else {
      callback(null, { endpoint, status: res.statusCode });
    }
  }).on('error', (err: any) => {
    callback(Error(err));
  });
}
