import https from 'https';
import Hatchet from '../src';
import { AdminClient } from '../src/clients/admin/admin-client';

const hatchet = Hatchet.init(
  {
    log_level: 'OFF',
  },
  {},
  {
    // This is needed for the local certificate in the example, but should not be used in production
    httpsAgent: new https.Agent({
      rejectUnauthorized: false,
    }),
  }
);

const admin = hatchet.admin as AdminClient;

admin.list_workflows().then((res) => {
  res.data.rows?.forEach((row) => {
    console.log(row);
  });
});
