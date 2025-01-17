import { createRoute } from "../../lib/api";
import { AppConfig } from "../../lib/schema";
import { getAppEndpoint, getDataDomain } from "../../lib/domains";
import { getEeConnection } from "../../lib/server/ee";
import { isEEAvailable } from "./ee/jwt";
import { isFirebaseEnabled, requireFirebaseOptions } from "../../lib/server/firebase-server";
import { nangoConfig } from "../../lib/server/oauth/nango-config";
import { isTruish } from "../../lib/shared/chores";

export default createRoute()
  .GET({ result: AppConfig, auth: false })
  .handler(async ({ req }) => {
    const publicEndpoints = getAppEndpoint(req);
    const dataHost = getDataDomain(publicEndpoints);

    return {
      docsUrl: process.env.JITSU_DOCUMENTATION_URL || "https://docs-jitsu-com.staging.jitsu.com/",
      websiteUrl: process.env.JITSU_WEBSITE_URL || "https://next.jitsu.com",
      credentialsLoginEnabled: !!process.env.TEST_CREDENTIALS && !!process.env.TEST_CREDENTIALS_SHOW_LOGIN,
      ee: {
        available: isEEAvailable(),
        host: isEEAvailable() ? getEeConnection().host : undefined,
      },
      disableSignup: process.env.DISABLE_SIGNUP === "true" || process.env.DISABLE_SIGNUP === "1",
      auth: isFirebaseEnabled()
        ? {
            firebasePublic: requireFirebaseOptions().client,
          }
        : undefined,
      billingEnabled: isEEAvailable(),
      syncs: {
        enabled: isTruish(process.env.SYNCS_ENABLED),
        scheduler: {
          enabled: !!process.env.GOOGLE_SCHEDULER_KEY,
          provider: process.env.GOOGLE_SCHEDULER_KEY ? "google-cloud-scheduler" : undefined,
        },
      },
      jitsuClassicUrl: process.env.JITSU_CLASSIC_URL || "https://cloud.jitsu.com",
      telemetry: {
        enabled: !!process.env.TELEMETRY_HOST,
        host: process.env.TELEMETRY_HOST === "__self__" ? publicEndpoints.baseUrl : process.env.TELEMETRY_HOST,
        writeKey: process.env.TELEMETRY_WRITE_KEY,
      },
      publicEndpoints: {
        protocol: publicEndpoints.protocol,
        host: publicEndpoints.hostname,
        cname: process.env.CNAME || "cname.jitsu.com",
        dataHost,
        port: publicEndpoints.isDefaultPort ? undefined : publicEndpoints.port,
      },
      logLevel: (process.env.FRONTEND_LOG_LEVEL || process.env.LOG_LEVEL || "info") as any,
      nango: nangoConfig.enabled
        ? {
            publicKey: nangoConfig.publicKey,
            host: nangoConfig.nangoApiHost,
          }
        : undefined,
    };
  })
  .toNextApiHandler();
