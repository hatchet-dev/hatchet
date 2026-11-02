# Company Background

Hatchet Technologies, Inc. ("Hatchet") builds an orchestration platform for AI agents, durable workflows, and background tasks. The core engine is open source under the MIT license and has more than 450,000 active instances; the same engine code powers both self-hosted deployments and Hatchet Cloud, which adds the managed control plane, multi-tenant operations, and hosted infrastructure. Hundreds of engineering teams worldwide run production workloads on Hatchet Cloud, which processes billions of tasks per month. Hatchet Cloud is the system under examination. This description covers Hatchet Cloud for the period July 28, 2026 to October 28, 2026.

Hatchet processes standard personal information (names, emails, addresses), protected health information (PHI) under HIPAA as a Business Associate, and personal data of EU residents subject to the General Data Protection Regulation (GDPR). Hatchet maintains compliance with SOC 2 (Security), HIPAA, and GDPR, and publishes its compliance posture at its Trust Center (trust.hatchet.run). The CTO is the designated Security Official and develops, implements, and oversees the information security program.

# Services Provided

Hatchet Cloud is a managed durable execution engine for background tasks, durable workflows, and AI agents. Every task and agent invocation is persisted so that work survives crashes and can be retried, replayed, and debugged.

Platform capabilities include configurable retry policies (including exponential backoff), cron scheduling, and future-dated execution. Multi-step durable workflows checkpoint state to a durable event log and survive crashes without losing progress. AI agent execution supports human-in-the-loop waits and real-time streaming. The engine provides concurrency control, rate limiting, task priority, and worker affinity routing, plus real-time monitoring, alerting, and logging.

Customers define tasks as functions in their application code using native SDKs for Python, TypeScript, Go, and Ruby. Workers running in the customer's environment connect outbound to the engine over gRPC. Customers use the web console at cloud.hatchet.run, the REST API, and the gRPC API.

Hatchet Cloud runs data-plane regions on Amazon Web Services in the United States (Oregon and Northern Virginia), the European Union (Ireland), and Australia (Sydney), and EU customers can keep tenant data in the EU region. Customers on custom plans can add dedicated capacity, additional regions, or bring-your-own-cloud deployments. Customer workloads include data pipelines, AI agent loops, video encoding, GPU scheduling, and event-driven systems.

# Service Commitments and System Requirements

Hatchet commits to customers on the security of Hatchet Cloud. Those commitments are in the Cloud Service Agreement (Terms and Conditions, published at hatchet.run/policies/terms), the Privacy Policy (hatchet.run/policies/privacy), the Data Processing Agreement (hatchet.run/policies/dpa), and the Trust Center (trust.hatchet.run). The Cloud Service Agreement states that Hatchet will use commercially reasonable efforts to secure the Cloud Service from unauthorized access, alteration, or use and other unlawful tampering.

Hatchet sets internal system requirements from the Trust Services Criteria for Security, the only category in scope. Those requirements are the control objectives below. Later sections describe how they are implemented.

**Security commitments.** Hatchet commits to the following:

- Encrypt all data in transit using TLS (minimum TLS 1.2) for API and worker traffic, and encrypt all data at rest on production storage and database services using encryption provided by the underlying infrastructure.
- Enforce logical tenant isolation so that every request is authenticated and scoped to a single tenant.
- Restrict access to production infrastructure and customer data to authorized employees on a least-privilege basis, and require multi-factor authentication for administrative, infrastructure, and critical-service access.
- Deploy a web application firewall (WAF) and DDoS mitigation at the network edge.
- Run vulnerability scans across infrastructure, applications, and dependencies, and perform independent third-party penetration testing at least annually. Remediate findings on severity-based timelines.
- Maintain an incident response framework covering detection, containment, eradication, recovery, and breach notification, including HIPAA notification to covered entities within required timeframes.
- Manage infrastructure and configuration changes through infrastructure-as-code and version-controlled code review, with segregation of duties between change authors and approvers.
- Review access at least annually, revoke access on termination, and disable dormant accounts.
- Assess third-party vendors before granting access to production environments or customer data, and keep an inventory of vendors by data sensitivity and risk.
- Publish security commitments and service descriptions through the Trust Center, Terms and Conditions, and Privacy Policy.

**System requirements.** The following requirements are how those commitments are met. Each one is implemented by a procedure described later in this section.

- Every user and service account has a unique identity. Hatchet uses Google Workspace single sign-on where the service supports it, and requires multi-factor authentication on critical systems. Access is least privilege. A manager and the Security Official approve grants. Unused accounts are removed within 30 days. Access is revoked within 24 hours of departure. Access is reviewed at least annually.
- Production access uses short-lived, audited privileged-access credentials issued by Teleport. Engineers do not hold standing production credentials.
- Production and infrastructure changes go through version control and an independent review. Emergency changes are documented within 1 business day.
- Production customer data has nightly encrypted backups retained for at least 3 days, point-in-time recovery where the database supports it, an annual restore test, and documented recovery time and recovery point objectives.
- Vulnerability remediation targets are 7 days for critical (P0), 30 days for high (P1), 60 days for medium (P2), and as needed for low (P3). An independent firm performs a penetration test at least annually.
- Audit logs cover authentication, administrative actions, privilege changes, and lifecycle events. Logs are sent to a central store and protected from tampering. Security logs are retained for 1 to 5 years. PHI access logs are retained for at least 6 years.
- Employees complete security awareness training within 30 days of start and annually after that, plus HIPAA and GDPR training where the role requires it.
- A vendor risk review is completed before a vendor connects to production. Vendors that handle sensitive data must provide a SOC 2 or ISO 27001 report, or an equivalent. The vendor inventory is reviewed annually.
- Workstations use disk encryption, a host firewall, a screen lock at 20 minutes, and anti-malware. PHI is not stored on local drives.

# Components of the System

The system is the infrastructure, software, people, data, and procedures Hatchet uses to deliver Hatchet Cloud and meet its security commitments.

The system is comprised of the following components:

- **Infrastructure.** The physical and virtual resources that support the environment, including cloud facilities, servers, storage, data storage devices, and the networks that connect them. Hatchet does not operate its own data centers.
- **Software.** The application programs and supporting system software, including the in-house platform, databases, and the external-facing web application. There is no native mobile application.
- **People.** The individuals involved in governance, operation, security, and use of the system, including executives, engineers, contractors, and customer users. Customers use the service but do not operate it.
- **Data.** The data the system uses and stores, including workflow and execution data, observability data, account metadata, and audit logs.
- **Procedures.** The automated and manual procedures by which the service is authorized, performed, and delivered, and by which security controls are operated.

## Architecture and Data Flow

Hatchet Cloud has two tiers. Both run on Amazon Web Services.

The control plane runs on the production Amazon EKS cluster in us-west-2 (Oregon). It handles sign-up, organizations, tenant provisioning, and billing hooks. Console login (email and password, Google, GitHub, and enterprise SSO) is handled by PropelAuth BYO, which Hatchet runs as a self-hosted sidecar next to the control-plane API, with its own PostgreSQL database. Customer identity data stays in Hatchet's infrastructure. The control plane issues short-lived exchange tokens that the console uses to call a shard.

The data plane is a set of regional shards on Amazon EKS in us-west-2 (Oregon), us-east-1 (Northern Virginia), eu-west-1 (Ireland), and ap-southeast-2 (Sydney). Shared multi-tenant shards and single-tenant shards use the same deployment module. Each shard runs the open-source engine: a REST API, a gRPC dispatcher for workers, scheduler and controller processes, a RabbitMQ cluster for internal messaging (with NATS as an optional publish/subscribe layer on some shards), and PostgreSQL as the durable store. PostgreSQL for a shard runs on Ubicloud or on Amazon RDS. Shards export execution telemetry (OpenTelemetry spans, metrics, and logs) to ClickHouse Cloud over AWS PrivateLink.

Google Cloud Platform does not host the control plane or the shards. It holds the Cloud KMS root key used for application envelope encryption, stores production secrets in Secret Manager, and stores build artifacts in Artifact Registry. Terraform reads secrets from Secret Manager and injects them into Kubernetes at deploy time.

Customer traffic reaches Hatchet Cloud over two paths. Browser and REST API traffic to the console and API hostnames is proxied through Cloudflare, which provides DNS, WAF, DDoS mitigation, and TLS termination (minimum TLS 1.2), and is then forwarded to the ingress load balancer on the EKS cluster. Worker traffic uses gRPC: customer workers open outbound, TLS-encrypted gRPC connections directly to a per-shard AWS Network Load Balancer, where the engine terminates TLS with an automatically issued certificate, and authenticate with a tenant API token. Cloudflare serves DNS for the gRPC hostnames but does not proxy them. In both paths the connection is initiated by the customer; Hatchet does not connect into customer networks.

Hatchet engineers do not have direct access to production from the internet. To reach a Kubernetes cluster or a database, an engineer first connects to Hatchet's private network over Tailscale, then requests a short-lived credential from Teleport, which records the session. Production servers do not accept SSH or remote desktop connections from the internet.

Data at rest sits in five places. Shard PostgreSQL stores workflow definitions, run state, payloads, and API token metadata. A regional TimescaleDB database stores run history and task events for the console and the API. ClickHouse Cloud stores execution telemetry. Amazon S3 stores large payload offload, where that feature is enabled, and participates in backup durability. Control-plane PostgreSQL stores accounts, organizations, and billing metadata.

```mermaid
flowchart LR
  subgraph customer [Customer environment]
    devs[Developers and console users]
    workers[Workers running customer code with SDKs]
  end
  subgraph edge [Cloudflare edge]
    cf[DNS, WAF, DDoS, TLS termination]
  end
  subgraph aws [AWS]
    subgraph cp [Control plane on EKS us-west-2]
      cpapi[Control plane API and console]
      cpdb[(Control plane PostgreSQL)]
      propel[PropelAuth BYO sidecar, self-hosted]
    end
    subgraph shard [Regional shard on EKS]
      api[REST API]
      grpc[gRPC dispatcher]
      engine[Scheduler and controllers]
      mq[RabbitMQ]
      pg[(PostgreSQL on Ubicloud or RDS)]
    end
  end
  subgraph gcp [GCP supporting services]
    kms[Cloud KMS]
    sm[Secret Manager]
  end
  ch[(ClickHouse Cloud via PrivateLink)]
  ops[Hatchet engineers via Tailscale and Teleport]
  devs --> cf --> cpapi
  cpapi --> cpdb
  cpapi --> propel
  cpapi -->|exchange token| api
  devs --> cf --> api
  workers -->|outbound gRPC over TLS, API token| grpc
  api --> pg
  grpc --> engine --> pg
  engine --> mq
  engine -->|OTEL| ch
  api -->|envelope encryption| kms
  sm -.->|secrets at deploy| cp
  sm -.->|secrets at deploy| shard
  ops --> grpc
  ops --> pg
```

## Infrastructure

Production customer workloads run on Amazon Web Services. Google Cloud Platform provides key management, secret storage, and artifact storage. Cloudflare provides edge security. Additional providers cover internal tooling, observability, and network connectivity. All production data processing occurs in data center regions operated by the providers below. Hatchet does not operate its own physical data centers.

Employees use company-managed laptops with endpoint protection, disk encryption, and a host firewall. Remote access to internal and production networks goes through an encrypted mesh VPN.

| Provider | Purpose |
| --- | --- |
| Amazon Web Services (AWS) | Primary cloud hosting. Amazon EKS runs the control plane and the regional shards. Also used for RDS, S3, ECR, and GuardDuty. Data-plane regions in the United States (Oregon and Northern Virginia), the European Union (Ireland), and Australia (Sydney). The control plane runs in Oregon. |
| Google Cloud Platform (GCP) | Cloud KMS for the application encryption root key, Secret Manager for production secrets, and Artifact Registry for build artifacts. GCP does not run production compute or production databases. |
| Cloudflare | DNS, WAF, DDoS protection, TLS termination, and content delivery. |
| ClickHouse Cloud | Managed analytics database for customer observability data. Shards export OpenTelemetry data over AWS PrivateLink. |
| Ubicloud | Managed PostgreSQL for production customer shards and for the regional TimescaleDB run-history databases, in the United States, the European Union, and Australia. |
| Fly.io | Application hosting for internal tooling and services. |
| Vercel | Hosting for the public website, documentation, and frontend assets. |
| Tailscale | Encrypted mesh VPN for authenticated access to internal and production systems. |

## Software

The platform is a web and API application. There is no native mobile application. Customers use the web console (cloud.hatchet.run), the REST and gRPC APIs, and the SDKs.

| Application / Vendor | Purpose |
| --- | --- |
| Hatchet Engine *(in-house)* | Core orchestration engine: durable execution, task scheduling, workflow management, retry logic, and agent coordination. |
| Hatchet API *(in-house)* | External REST and gRPC APIs for client SDKs, worker connections, and platform management. |
| Hatchet Cloud Console *(in-house)* | Web application for workflow monitoring, debugging, configuration, and tenant administration. |
| Hatchet SDKs *(in-house)* | Client libraries for defining tasks, connecting workers, and calling the platform from application code. |
| Control plane *(in-house)* | Sign-up, organizations, tenant provisioning, billing hooks, and shard exchange tokens. Runs on the production EKS cluster. |
| PostgreSQL | Primary durable store for each shard (workflow definitions, run state, payloads, and API token metadata) and for the control plane (accounts, organizations, and billing metadata). Runs on Ubicloud or Amazon RDS. |
| RabbitMQ *(self-hosted)* | Internal message bus on each shard. Not exposed to customers. |
| NATS *(self-hosted)* | Optional internal publish/subscribe layer on shards where it is enabled, used for real-time event fan-out between engine processes. Not exposed to customers. |
| TimescaleDB *(self-hosted)* | Time-series extension on a dedicated PostgreSQL database per region that stores run history and task events for the console and the API. |
| PropelAuth BYO *(self-hosted)* | Licensed software Hatchet operates for console sessions, enterprise SSO (SAML and OIDC), SCIM, and passkeys. Customer identity data stays in Hatchet's infrastructure. |
| HyperDX *(self-hosted)* | Central store and dashboards for production metrics, logs, and traces, built on OpenTelemetry and ClickHouse. Used for operational monitoring and alerting. |
| GitHub | Version control, pull-request code review, CI, and Dependabot dependency scanning. |
| Google Workspace | Identity provider (SSO via Google sign-in), corporate email, and collaboration. |
| 1Password | Credential and secrets management for the team. |
| Teleport | Privileged access management for production infrastructure: short-lived credentials, session recording, and audited access. |
| Slack | Internal communication, access request workflow, and incident coordination. Also the destination for the optional, customer-configured tenant alerting integration. |
| Oneleet | Compliance platform for control monitoring, evidence collection, policy management, and risk assessment. |
| Oneleet Agent | Endpoint agent for disk encryption, firewall status, screen lock, and anti-malware on employee devices. |
| incident.io | Incident management and response coordination. |
| Sentry | Application error tracking and performance monitoring for production services. |
| Better Stack | Uptime monitoring, the public status page (status.hatchet.run), and incident alerting. |
| pganalyze | PostgreSQL performance monitoring and query analysis. |
| Postmark | Transactional email for customer-facing notifications. |
| GDPR Local | Intake for GDPR data subject access requests. |

## People

Hatchet is a fully remote team. The people involved in the system are organized into the following groups.

- **Executive management.** The CEO and the CTO, who are the co-founders. They set security strategy and risk tolerance, approve policies at least annually, and sit in the business continuity and disaster recovery escalation chain.
- **Security.** The CTO is the designated Security Official. The Security Official owns the information security program: incident response, risk assessments, approval of classified-media disposal, and evaluation of control effectiveness. Engineering carries out the day-to-day procedures.
- **Engineering and operations.** Engineers build the platform, operate production infrastructure, review code, respond to incidents, and perform change management and access provisioning. Production and customer-data access is limited to engineers with an operational need. Engineering includes contractors with access to production systems and data, who sign the same confidentiality agreements, acknowledge the same policies, complete the same training, and receive access under the same controls as employees.
- **Customers.** Customer administrators and developers use the console, the APIs, and the SDKs, and manage their own tenant configuration, API tokens, and team membership. They do not have access to Hatchet's internal systems or to other tenants' data. Customer responsibilities are listed under Complementary User Entity Controls.

## Data

Hatchet classifies data as **Public**, **Internal**, **Confidential**, or **Secret**. Handling, access, encryption, and disposal requirements increase at each level. All customer data is Confidential at minimum.

The system stores customer workflow data (definitions, execution parameters, input and output payloads, durable event logs), customer observability data (execution telemetry, metrics, alerting state, and logs), account and tenant metadata (identifiers, emails, authentication tokens, API keys, and configuration), and audit and security logs (authentication events, administrative actions, privilege changes, and lifecycle events). For customers with a signed Business Associate Agreement, Hatchet processes PHI as a Business Associate. PHI is Confidential. Encryption keys, service account credentials, and internal API tokens are Secret.

Data in transit uses TLS, with a minimum of TLS 1.2 and a preference for TLS 1.3. Cloudflare terminates TLS at the edge and uses full or strict encryption between the edge and origin. Connections from production services to databases, the message bus, and the observability platform are encrypted, and engineer access to internal systems goes over Tailscale. Data at rest, including production databases, object storage, and backups, is encrypted with the provider's managed encryption.

Retention follows the data type, the classification, and the applicable regulation. Customer data retention follows the customer contract. Workflow execution logs are retained for up to 90 days. Security logs are retained for 1 to 5 years. PHI access logs are retained for at least 6 years. Account data is retained while the account is active and for up to 12 months after closure. Backup copies are retained from 14 days to 1 year, depending on classification. At the end of the retention period, Hatchet deletes the data using methods consistent with NIST 800-88, including cryptographic erasure where the storage system supports it.

## Procedures

### Access Management

Access follows least privilege. Every user and service account has a unique identifier and only the access its function requires. Administrative duties use separate administrative accounts. Hatchet uses single sign-on through Google Workspace wherever the service supports it. Multi-factor authentication is required for critical services, infrastructure access, and administrative accounts, and is enforced on the cloud providers, the identity provider, source control, and collaboration tools. Services enforce session timeouts.

New access is requested through a designated request channel and needs approval from a manager and from the Security Official. Production administrative access, including databases and Kubernetes, is issued by Teleport as short-lived credentials. Sessions expire on their own. Access events are logged.

Access is reviewed at least annually across critical vendors and systems. Automated monitors check for dormant accounts, MFA, completed offboarding, and access reviews in the trailing 12 months. Accounts unused for 30 days are removed. When employment or a contractor engagement ends, Hatchet revokes access within 24 hours using an offboarding checklist. Shared accounts are prohibited. Service account credentials are rotated when an employee who knows the credential leaves.

### Customer-side Logical Access

Customers control who can use their own tenant. Console users sign in with email and password, Google, GitHub, or enterprise SSO. Enterprise SSO uses SAML or OIDC. SCIM provisioning is available to enterprise customers. Organization owners invite members and assign a role (owner, admin, or member) on each tenant.

API tokens are tenant-scoped JWTs. A tenant administrator creates them in the console. The token is shown once, can be given an expiry, and can be revoked. Workers authenticate with these tokens over TLS.

The tenant is the isolation boundary. Every API and gRPC request is authenticated and scoped to one tenant. Webhook and integration credentials are stored encrypted, as described under Key Management. The customer chooses the data-plane region when the tenant is created. For example, EU tenants stay in eu-west-1 (Ireland).

### Change Management

Code and infrastructure-as-code changes go through GitHub. Each change is a pull request. CI runs before merge, and at least one reviewer other than the author approves it. The author does not approve their own change.

Changes are tested in CI and by manual verification before production. Development, staging, and production are separate environments. Non-production environments do not hold customer data. Changes are classified by risk. High-risk changes get additional review, and the author or reviewer includes a rollback and test plan when the risk calls for one.

Emergency changes may be implemented before formal approval. Documentation and approval are completed within 1 business day. Changes outside version control, such as manual configuration in a third-party console, need an explicit approval in an approved channel before implementation.

Changes are logged. Significant changes get a post-implementation review. Customer-visible changes are communicated on the status page and in release notes, and internally to the teams affected.

### Secure Development

Application and infrastructure code live in separate GitHub repositories governed by the pull-request, CI, and independent-review rules in Change Management. For Terraform changes, the plan output is reviewed before apply. Dependabot reports vulnerable dependencies, and container images are scanned in Amazon ECR.

Security fixes for the open-source project are released for the current major version. Vulnerability reports for the open-source repositories go through the private advisory form on GitHub, as described in SECURITY.md, with security@hatchet.run as the fallback. SDK vulnerabilities are tracked; the remediation timelines in Vulnerability Management apply to the hosted service rather than to SDK upgrades that customers choose when to install. Secure coding and common web application weaknesses are covered in the annual security training.

### Data Communications and Network Security

Network access to production uses layered controls. Security groups and firewall rules block public access to non-essential ports, including SSH and remote desktop, and databases are not publicly reachable. Object storage blocks public access unless there is a documented business justification. Automated monitors check these settings on AWS, GCP, and Cloudflare.

Console and API traffic passes through Cloudflare for WAF, DDoS mitigation, DNSSEC where the domain supports it, and HTTPS. Engineer access to internal and production systems goes over Tailscale, which authenticates each device and encrypts traffic. Unauthorized Tailscale peers are flagged for removal. Production database access requires unique credentials, and database connections are encrypted.

### Key Management

Data stored in production databases, object storage, and backups is encrypted at rest using the hosting provider's managed encryption.

Sensitive values the application stores on behalf of customers, such as third-party credentials and signing secrets, are additionally encrypted by the application before they are written to the database. The application uses envelope encryption: each value is encrypted with its own data key, and data keys are protected by a root key held in Google Cloud KMS that Hatchet does not export. The root key rotates automatically on a fixed schedule. The keys used to sign customer API tokens are protected by the same root key.

Infrastructure secrets are stored in Google Secret Manager and delivered to workloads at deploy time; they are not kept in source control or on employee devices. Team credentials are kept in a shared password manager. Access to Cloud KMS and Secret Manager is limited to the production workload identity and to the engineers who operate infrastructure.

Credentials known to a departing employee are rotated at offboarding. TLS certificates for customer-facing endpoints and for services inside the cluster are issued and renewed automatically.

### Logging and Monitoring

Production systems emit audit logs for login attempts, administrative actions, privilege changes, and major application lifecycle events. Logs are transmitted to a central store and protected from tampering. GCP audit logs are enabled for Cloud KMS and Secret Manager.

Production services emit metrics, structured logs, and distributed traces through OpenTelemetry to a central observability platform that Hatchet operates. Engineers use it to monitor the health of every shard and the control plane, including request latency and error rates, queue depth, worker connectivity, database load and capacity, and message-bus health. Application errors are captured in an error-tracking service, and external uptime checks run against the API and the console from outside Hatchet's network. Alerts on these signals are classified as critical or non-critical. Critical alerts page the on-call engineer; non-critical alerts are routed to the engineering team for review during working hours. A critical alert that indicates customer impact follows the incident response process below and is reflected on the public status page where appropriate.

Security monitoring runs alongside operational monitoring. AWS GuardDuty performs threat detection in the production accounts, attack surface scanning runs continuously against company domains, and high-risk audit events are reviewed. A suspected security incident starts the incident response process below.

### Vulnerability Management

Vulnerabilities are found by automated scanning of production infrastructure (AWS GuardDuty and attack surface scanning), by dependency and container image scanning (GitHub Dependabot and Amazon ECR image scanning), and by a third-party penetration test at least annually.

Once a vulnerability is accepted, remediation targets are 7 days for critical (P0), 30 days for high (P1), 60 days for medium (P2), and as needed for low (P3). Monitors track ECR image findings, GuardDuty findings, attack surface findings, Dependabot alerts, and penetration test findings. Patches are tested outside production before they are deployed. Actively exploited vulnerabilities follow an accelerated path with testing and approval. Findings are retained for 5 years.

### Endpoint Security

Employee devices run the Oneleet Agent, which checks disk encryption, host firewall, screen lock, and anti-malware. The Workstation Security Policy requires the screen lock to engage after 20 minutes. Monitors flag devices that are dormant or out of compliance.

Employees reach internal and production networks through Tailscale. PHI and other sensitive data stay on approved production systems, not on local drives, unless there is explicit authorization and encryption. A lost or stolen device is reported immediately and deactivated.

### Physical Security

Hatchet is a fully remote company. Some employees also use a co-working space. Remote employees use encrypted home networks, store devices securely when not in use, and prevent unauthorized viewing of sensitive information.

In a co-working space, Hatchet relies on the provider's perimeter controls (keycards, entry systems, and surveillance) and adds clear-desk and clear-screen rules, visitor escort, and locked storage for sensitive materials.

Hatchet does not operate data centers. Physical security of production infrastructure belongs to the cloud providers in the Infrastructure section. Hatchet reviews each provider's physical security documentation annually.

### Incident Response

Incident response has six phases: preparation, identification, containment, eradication, recovery, and lessons learned. Severity is Critical, High, Medium, or Low, with an escalation path for each.

Employees report suspicious activity immediately. The response team coordinates in a dedicated incident management tool, and customer communication goes through the public status page. Where an incident affects customer data, Hatchet notifies affected customers within the timeframes required by contract and by applicable regulation, including HIPAA breach notification to covered entities.

After resolution, Hatchet reviews root cause and the response, and records the incident. Hatchet also runs incident response exercises.

### Backups and Disaster Recovery

Every production database that holds customer data is backed up automatically. Nightly full backups are retained for at least 3 days, and point-in-time recovery is enabled on databases that support it. Backups are kept longer, from 14 days to 1 year, where data classification or a regulatory obligation requires it. Backups are encrypted at rest and in transit, and only authorized engineers can access them. Objects in Amazon S3 are protected by the provider's built-in durability. The infrastructure team owns the recovery runbook.

Backup restoration is tested at least annually, and the results are recorded. Disaster recovery plans set a recovery time objective and a recovery point objective for each critical system, tie into incident response, and are reviewed annually or after a significant change. Engineers with a role in recovery receive training annually.

### Vendor Management

Hatchet keeps an inventory of third-party vendors, including the service each provides, the data each can access, and the risk level. A vendor is connected to production only after leadership approval and a vendor risk review. The review covers the vendor's security program, compliance posture (SOC 2, ISO 27001, or an equivalent), data segregation, and support for deletion. Vendors that handle sensitive data must show an established security program. Vendors with potential access to PHI sign a Business Associate Agreement. Agreements state the security expectations and service commitments. The inventory and the risk reviews are updated at least annually.

### Email Security

Hatchet publishes DMARC on its domains. Google Workspace's advanced protections against malicious attachments, links, and spoofed senders are enabled. Super-administrator access is restricted, and mailbox access is not delegated to non-admin users.

### Policy Inventory

The policies below are the current published versions in the compliance platform. Each line gives the version and the date that version was published. Current employees sign each policy.

- Acceptable Use Policy, v2.1, published July 21, 2026.
- Access Management Policy, v3.1, published July 21, 2026.
- Asset Management Policy, v2.0, published July 21, 2026.
- Audit Log Management Policy, v3.0, published July 23, 2026.
- Business Continuity and Disaster Recovery Policy, v2.0, published July 21, 2026.
- Change Management Policy, v2.1, published July 21, 2026.
- Code of Conduct, v2.0, published July 21, 2026.
- Data Backup and Recovery Policy, v2.0, published July 21, 2026.
- Data Management, Retention, and Destruction Policy, v4.1, published August 7, 2026.
- Employee Privacy Notice, v1.0, published July 27, 2026.
- Human Resource Security Policy, v2.0, published July 21, 2026.
- Incident Response Policy, v3.0, published July 21, 2026.
- Law Enforcement and Compelled Disclosure Policy, v1.0, published July 27, 2026.
- Password Policy, v2.0, published July 21, 2026.
- Physical Security Policy (Coworking), v1.0, published July 21, 2026.
- Physical Security Policy (Remote), v2.0, published July 21, 2026.
- Risk Management Policy, v2.0, published July 21, 2026.
- Vendor Management Policy, v2.0, published July 21, 2026.
- Vulnerability Management Policy, v3.0, published July 21, 2026.
- Whistleblower Policy, v3.0, published July 21, 2026.
- Workstation Security Policy, v3.0, published July 21, 2026.

## System Boundaries

The components above are the system under examination. The system includes everything that delivers Hatchet Cloud, processes or stores customer data, or supports a control behind a security commitment. Security tools that support those controls, including the WAF, VPN, endpoint agent, threat detection, vulnerability scanning, and log management, are in scope. In-scope third parties that are not carved-out subservice organizations (such as Teleport, Postmark, Vercel, and PropelAuth) are explained under Subservice Organizations.

The following are outside the system. They do not process or store customer production data, they are not required to deliver the service, and they do not support controls behind the security commitments:

- Corporate back-office tools for finance, payroll, banking, equity, legal, sales, and scheduling.
- Internal productivity and communication tools, including Notion, Linear, Figma, Discord, Zoom, and AI-assisted development tools.
- Autumn, PostHog, Pylon, Stripe, and Railway, which touch the customer relationship or internal operations but not the production data path, as described below.

A few of these are close enough to the product that their exclusion needs explaining. Autumn (plan entitlements) and Stripe (payments) handle billing; neither is consulted when a task runs, and neither stores workflow data. Pylon is the customer support desk and holds only what customers put in support conversations. PostHog receives product-usage analytics from the console, including the user's name and email and session recordings with inputs masked; tenants can opt out, and it never receives workflow data. Railway hosts internal Hatchet automation, such as growth and marketing workers. No Hatchet Cloud production service runs on Railway, and it does not store customer production data.

Third-party connectivity uses authenticated APIs, SSO, the VPN, and service-specific access controls. Vendors with access to in-scope systems are under contract, and that access follows the least-privilege and monitoring controls in Procedures. Customer-managed environments, including customer application code, worker runtimes, and self-hosted Hatchet deployments, are outside the system.

# Control Environment

## Integrity and Ethical Values

The Code of Conduct sets expectations for employees, contractors, interns, and anyone working with Hatchet in a professional capacity. It covers respect and inclusion, privacy and security, professionalism, collaboration, and reporting. Employees acknowledge the Code during onboarding; contractors acknowledge it before they start.

The Whistleblower Policy lets employees report harassment, discrimination, policy violations (including HIPAA), and other misconduct anonymously and without retaliation. Reports go through an anonymous reporting channel. The CTO reviews them. A report about the CTO goes to the CEO.

A disciplinary process covers security violations and policy non-compliance. The Security Official investigates, decides the consequence (up to termination), acts to prevent a repeat, and keeps the investigation record for at least 7 years. Employees who report in good faith are protected from intimidation, threats, and retaliation.

## Management Philosophy and Operating Style

Security controls sit in the development and operations workflow. Infrastructure-as-code, CI, mandatory code review with segregation of duties, and continuous compliance monitoring are the main enforcement mechanisms.

The CEO and the CTO both take part in security governance. They review and approve security policies at least annually, and both are in the business continuity and disaster recovery escalation chain. They set risk tolerance.

## Organizational Structure and Assignment of Authority

Hatchet does not have a formal board of directors. Oversight of the security program sits with the CEO and the CTO, who review and approve all policies and the annual risk assessment. The division of responsibilities among executive management, the Security Official, and engineering is described under People.

Roles are defined in the information security program and covered in onboarding. Policy acknowledgment, training records, and access that matches job function keep those assignments in place.

## Commitment to Competence

Hiring for roles that access sensitive data or production systems includes a background or reference check scaled to the role, as described under HR Policies and Practices. Performance reviews cover security practices for roles with significant system or data access.

Employees complete security awareness training within 30 days of start and annually after that. The training covers social engineering, workstation security, common attack vectors, and handling sensitive data. Employees whose roles require it also complete HIPAA training (PHI types, obligations, and penalties) and GDPR training (personal data types, principles, and data subject rights). Completion is tracked, and overdue training is flagged.

## HR Policies and Practices

The Human Resource Security Policy covers the employment and contractor lifecycle. HR security records are retained for 7 years.

- **Hiring.** Background or reference checks are completed before access to sensitive data or production systems, scaled to the role. New employees sign a non-disclosure agreement before they access company data or systems. A limited-scope exemption from a background check requires documented approval from the CEO or the CTO and still requires a confidentiality agreement.
- **Onboarding.** Onboarding follows a checklist. New employees receive role-appropriate access, complete the training described under Commitment to Competence, and acknowledge applicable policies. A monitor checks that the checklist is completed.
- **During employment.** Employees acknowledge applicable security policies, and acknowledgments are tracked. Role-based and compliance training follows the person's responsibilities. Violations go through the disciplinary process.
- **Offboarding.** Departures follow a checklist. Access is revoked within 24 hours, as described under Access Management. Company property is collected, and departing employees are reminded that confidentiality obligations continue. A monitor checks that the person has been removed from connected systems.

# Risk Assessment

Hatchet runs a formal risk assessment at least annually. The Risk Management Policy scores each risk by likelihood and impact (financial, operational, reputational, and regulatory) and rates it high, medium, or low.

High residual risks, such as regulatory non-compliance or a significant data exposure, are treated first. Medium risks have a documented contingency plan. Low risks are monitored. A risk above tolerance is escalated to executive management.

Each risk has an owner and a treatment in the risk register: mitigate, avoid, transfer, or accept, with the responsible party, the resources, and a deadline. Fraud risk is assessed as part of the same process. Control effectiveness is reviewed annually. Third-party and vendor risk is assessed before a vendor connects to production and reviewed annually, as described under Vendor Management.

# Information and Communication

## Internal Communication

Security policies live in the compliance platform. Employees acknowledge the policies that apply to them, and unsigned policies are flagged. New employees receive policies and training during onboarding. Slack is used for access requests, incident coordination, and change notices. Findings from risk assessments, control monitoring, and internal audits go to executive management. Policy changes are communicated internally, and employees re-acknowledge the updated policy.

## External Communication

The Trust Center, the Terms and Conditions, the Privacy Policy, and the Data Processing Agreement publish Hatchet's security commitments, compliance posture, and service terms at the URLs listed under Service Commitments and System Requirements. The status page (status.hatchet.run) shows availability and incident history for the API and the console, and maintenance that affects customers is posted there and sent as external notices.

Vulnerability reports on the open-source repositories follow SECURITY.md, as described under Secure Development; reports on the hosted service go to security@hatchet.run. Hatchet does not operate a bug bounty. Privacy requests go to legal@hatchet.run. Requests from law enforcement and other compelled-disclosure demands also go to legal@hatchet.run, are handled under the Law Enforcement and Compelled Disclosure Policy, and are preserved for 90 days where that policy requires it.

# Monitoring

## Ongoing Automated Monitoring

The compliance platform connects to the cloud accounts, the identity provider, source control, and employee devices, and runs automated checks continuously. The checks cover the controls described in Procedures: user access and offboarding, MFA, network exposure and firewall configuration, endpoint protection, vulnerability and dependency findings, code review enforcement, audit log configuration, TLS settings, and email security. A failing check opens a finding that is assigned and tracked to closure.

## Periodic Evaluations

Hatchet performs an internal security audit annually. The risk assessment is annual, and the risk register is updated when something material changes between assessments. Access reviews are completed at least annually. The CEO and the CTO review and approve security policies annually.

## Subservice Organization Monitoring

Hatchet reviews each subservice organization's SOC 2 report, or an equivalent independent assessment, at least annually, monitors their status pages for disruptions, and raises issues through the provider's support channels. Where a provider supports it, the compliance platform also checks Hatchet's configuration of that provider continuously; for providers it cannot connect to, Hatchet verifies access and network exposure directly as part of the periodic review.

## Deficiency Remediation

A control gap found by a monitor, an audit, or a management review is logged, assigned, and tracked to closure. Vulnerability findings follow the severity-based timelines described under Vulnerability Management. Penetration test findings are tracked individually and verified when closed. Open items are escalated to executive management.

# Subservice Organizations

The description does not extend to the services provided by Amazon Web Services, Google Cloud Platform, Cloudflare, ClickHouse Cloud, and Ubicloud (collectively, the subservice organizations). Section 4 of this report and the description of the system only cover the relevant trust services criteria and related controls in support of the achievement of Hatchet's service commitments and system requirements and exclude the related controls of the subservice organizations.

Although the subservice organizations have been carved out for the purposes of this report, Hatchet's management has assumed, in the design of the system, that certain complementary subservice organization controls (CSOCs) would be implemented by the subservice organizations. Such controls are necessary, in combination with controls at Hatchet, to provide reasonable assurance that Hatchet's service commitments and system requirements were achieved. Because the related service commitments and system requirements can only be achieved if the CSOCs are suitably designed and operating effectively during the audit period, each user entity must evaluate Hatchet's controls, related tests of controls, and results of tests described in Section 4 of this report, considering the types of related CSOCs expected to be implemented at the subservice organizations as shown below.

The subservice organizations and their roles are:

- **Amazon Web Services.** Cloud hosting for the control plane, the shards, and production data stores.
- **Google Cloud Platform.** Key management, secret storage, and build artifact storage.
- **Cloudflare.** DNS, WAF, DDoS mitigation, and TLS termination for console and API traffic.
- **ClickHouse Cloud.** Managed analytics database for customer observability data.
- **Ubicloud.** Managed PostgreSQL for production shards and run-history databases.

Each is described further under Infrastructure, and management's monitoring of these providers is described under Monitoring. The following table lists the complementary subservice organization controls that each is expected to implement.

| Subservice Organization | Complementary Subservice Organization Control |
| --- | --- |
| Amazon Web Services | Restrict physical access to data center facilities to authorized personnel. |
| Amazon Web Services | Implement environmental protections (fire detection and suppression, climate control, and power redundancy) to safeguard infrastructure from environmental threats. |
| Amazon Web Services | Securely dispose of or sanitize storage media containing customer data before reuse or decommissioning. |
| Amazon Web Services | Restrict logical access to production host infrastructure, hypervisors, and management planes to authorized personnel. |
| Amazon Web Services | Monitor cloud infrastructure for security events, anomalies, and unauthorized access attempts. |
| Amazon Web Services | Encrypt data at rest on managed storage and database services using industry-standard encryption algorithms. |
| Amazon Web Services | Detect and mitigate network-level and application-level attacks against cloud infrastructure. |
| Google Cloud Platform | Restrict physical access to data center facilities that host Cloud KMS and Secret Manager to authorized personnel. |
| Google Cloud Platform | Implement environmental protections to safeguard that infrastructure from environmental threats. |
| Google Cloud Platform | Securely dispose of or sanitize storage media before reuse or decommissioning. |
| Google Cloud Platform | Restrict logical access to Cloud KMS key material and cryptographic operations to authorized services and personnel. |
| Google Cloud Platform | Restrict logical access to Secret Manager secrets to authorized services and personnel. |
| Google Cloud Platform | Encrypt Cloud KMS key material and Secret Manager payloads at rest. |
| Google Cloud Platform | Monitor Cloud KMS and Secret Manager for security events, anomalies, and unauthorized access attempts. |
| Cloudflare | Maintain WAF rule sets and DDoS mitigation controls to protect customer traffic traversing the edge network. |
| Cloudflare | Enforce TLS termination and encrypt data in transit across edge infrastructure using current protocol versions. |
| Cloudflare | Restrict physical and logical access to network edge infrastructure to authorized personnel. |
| Cloudflare | Monitor edge network infrastructure for security events, anomalies, and unauthorized access attempts. |
| ClickHouse Cloud | Restrict physical and logical access to managed database infrastructure to authorized personnel. |
| ClickHouse Cloud | Encrypt customer data at rest on managed storage using industry-standard encryption algorithms. |
| ClickHouse Cloud | Monitor database infrastructure for security events and unauthorized access attempts. |
| ClickHouse Cloud | Maintain backup and recovery controls for customer data stored in managed databases. |
| Ubicloud | Restrict physical access to data center facilities to authorized personnel. |
| Ubicloud | Restrict logical access to host and database infrastructure to authorized personnel. |
| Ubicloud | Encrypt customer data at rest using industry-standard encryption algorithms. |
| Ubicloud | Maintain backup and recovery controls for customer data stored in managed databases. |
| Ubicloud | Monitor database infrastructure for security events and unauthorized access attempts. |

Teleport, Postmark, Vercel, and PropelAuth are third parties, not subservice organizations. Teleport issues short-lived production credentials for Hatchet engineers; customers do not use it, and Hatchet's access commitments are met by Hatchet's own controls. Postmark delivers transactional notification email, and Hatchet's security commitments do not depend on a control that only Postmark performs. Vercel hosts the public website and frontend assets, which are classified Public and are not in the customer data path for the orchestration service. PropelAuth licenses the authentication software that Hatchet runs inside its own infrastructure; the software performs a license check against PropelAuth, but no customer or identity data is sent to PropelAuth, and it is neither a subservice organization nor a subprocessor.

# System Incidents

## A. Incidents Pertaining to the Audit Period

No significant security incidents have occurred during the audit period that would have a material effect on the suitability of the design and operating effectiveness of the controls and/or description of the system covered in Section 3 of this report.

## B. Incidents Subsequent to the Audit Period

Management is not aware of any incidents that occurred subsequent to the period covered by Section 3 of this report through the date of the service auditor's report that would have a significant effect on management's assertion and description of its system.

# Significant Changes During the Audit Period

## A. Significant Changes and Events During the Audit Period

No significant changes and events have occurred during the audit period that would have a material effect on the suitability of the design and operating effectiveness of the controls and/or description of the system covered in Section 3 of this report.

## B. Significant Changes and Events Subsequent to the Audit Period

Subsequent to the audit period, Hatchet began removing the TimescaleDB extension as a dependency of Hatchet Cloud. Run history and task events will be stored in standard PostgreSQL on the same managed database providers, under the same access, encryption, backup, and monitoring controls described in this section. This change affects a component described in this section but not the design or operation of Hatchet's controls.

Other than the change noted above, management is not aware of any changes that occurred subsequent to the period covered by Section 3 of this report through the date of the service auditor's report that would have a significant effect on management's assertion and description of its system.

# Applicable Trust Services Criteria

All Security criteria are applicable to the system. The Availability, Processing Integrity, Confidentiality, and Privacy categories are not included within the scope of this examination.

# Complementary User Entity Controls

Hatchet Cloud is a shared-responsibility service, and Hatchet's controls do not cover the customer's environment. The complementary user entity controls (CUECs) below are the customer responsibilities assumed in the design of Hatchet's controls. They are not an exhaustive list; each customer's auditor should evaluate what additional controls apply in the customer's environment.

| Trust Services Criteria | Complementary User Entity Controls |
| --- | --- |
| CC2.1 | User entities are responsible for the security and integrity of their own data, including data used by Hatchet systems and services. |
| CC6.1 | User entities protect API tokens and worker credentials, rotate them when personnel change, and set an expiry where the token supports one. |
| CC6.1 | User entities enable SSO or multi-factor authentication on the identity provider they use to sign in to the console. |
| CC6.2 | Authorized personnel at the user entity determine which individuals need access to Hatchet and grant permissions accordingly, including managing any API keys or credentials used to access the platform. |
| CC6.3 | User entities periodically review authorized users and their access privileges. |
| CC6.6 | User entities are responsible for protecting their data in transit to Hatchet. |
| CC6.6 | User entities maintain physical security and environmental controls for devices and access points, including those used by remote personnel, that connect to Hatchet. |
| CC6.6 | User entities secure the worker runtime where their code and task payloads execute. |
| CC6.7 | User entities do not place PHI in Hatchet without an executed Business Associate Agreement, and they avoid placing secrets in task payloads or logs. |
| CC7.2 | User entities monitor the status page, configure tenant alerting, and verify signatures on outbound webhooks. |
| CC9.2 | User entities review this report, and the subservice organizations' reports, as part of their own vendor management program. |
