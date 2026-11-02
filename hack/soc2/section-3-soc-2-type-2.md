# Company Background

Hatchet Technologies, Inc. ("Hatchet" or "the Company") is a developer infrastructure company founded in 2023 by Alexander Belanger and Gabriel Ruttner. The Company builds an orchestration engine for distributed workloads — a platform that enables engineering teams to schedule, execute, observe, and manage background tasks, durable workflows, and AI agent workloads at scale. Hatchet participated in the Y Combinator Winter 2024 batch and is headquartered in New York City.

As of the examination period, Hatchet employs eight employees and one contractor, for a total of nine active personnel. The founding team brings deep experience in distributed systems and infrastructure engineering, which directly informs the Company's product architecture and security program. The Company operates in the infrastructure and developer tools space, serving customers in the United States and the European Union.

Hatchet processes standard personal information (names, emails, addresses), protected health information (PHI) under HIPAA as a Business Associate, and personal data of EU residents subject to the General Data Protection Regulation (GDPR). The Company maintains compliance with SOC 2 (Security), HIPAA, and GDPR, and publishes its compliance posture and documentation at its Trust Center (trust.hatchet.run).

The Hatchet platform is open source under the MIT license, available both as a self-hosted deployment and as a fully managed cloud service (Hatchet Cloud). The open-source codebase and the cloud service share the same code, so security improvements apply uniformly across both deployment models.

Security at Hatchet is owned at the executive level. Gabriel Ruttner, the Company's CTO and co-founder, serves as the designated Security Official and is responsible for developing, implementing, and overseeing the information security program.

# Services Provided

Hatchet provides a managed orchestration platform that enables engineering teams to build and operate distributed workloads — specifically background tasks, durable workflows, and AI agents. The platform functions as a durable execution engine: every task invocation is persisted, enabling automatic retries, replays, debugging, and complex workflow patterns that traditional queuing systems do not support.

Core platform capabilities include task orchestration with configurable retry policies (including exponential backoff), cron scheduling, and future-dated execution; multi-step durable workflows that checkpoint state to a durable event log and survive crashes without losing progress; AI agent execution with human-in-the-loop waits and real-time streaming; first-class concurrency control, rate limiting, task priority, and worker affinity routing; and real-time monitoring, alerting, and logging for all tasks and workflows.

Hatchet supports applications written in Python, TypeScript, Go, and Ruby through native SDKs. Customers integrate by defining tasks as functions in their application code and connecting workers to the Hatchet engine via gRPC.

**Hatchet Cloud** — the fully managed service — is the system under examination in this report. Hatchet Cloud operates data plane regions on Amazon Web Services in the United States (Oregon, Northern Virginia), the European Union (Ireland), and Australia (Sydney). Customers on custom plans can add dedicated capacity, additional regions, or bring-your-own-cloud deployments.

Hatchet's customers are primarily engineering teams at technology companies running workloads such as data pipelines, AI agent loops, video encoding, GPU scheduling, and event-driven architectures. The Company serves customers across the United States and the European Union.

# Service Commitments and System Requirements

Hatchet makes specific commitments to customers regarding the security of the Hatchet Cloud platform. The Company communicates these commitments through the Cloud Service Agreement (Terms and Conditions, published at hatchet.run/policies/terms), Privacy Policy (hatchet.run/policies/privacy), Data Processing Agreement (hatchet.run/policies/dpa), and the published Trust Center (trust.hatchet.run). The Cloud Service Agreement commits that Hatchet will use commercially reasonable efforts to secure the Cloud Service from unauthorized access, alteration, or use and other unlawful tampering.

Hatchet establishes internal system requirements based on the applicable Trust Services Criteria for the Security category — the sole category in scope for this examination. These requirements translate the customer-facing commitments into specific control objectives that the Company designs, implements, and monitors.

**Security** commitments include, but are not limited to, the following:

- Encrypting all data in transit using TLS (with a minimum of TLS 1.2) for API and worker traffic, and encrypting all data at rest on production storage and database services using encryption provided by the underlying cloud infrastructure.
- Enforcing logical tenant isolation so that all requests are authenticated and scoped to a single tenant, preventing cross-tenant data access.
- Restricting access to production infrastructure and customer data to authorized personnel on a least-privilege basis, with multi-factor authentication required for all administrative, infrastructure, and critical-service access.
- Deploying a web application firewall and DDoS mitigation at the network edge to protect the platform from external attacks.
- Conducting regular vulnerability scans across infrastructure, applications, and dependencies, and performing independent third-party penetration testing at least annually, with findings remediated according to severity-based timelines.
- Maintaining an incident response framework with defined procedures for detection, containment, eradication, recovery, and breach notification — including HIPAA-specific notification to covered entities within required timeframes.
- Managing all infrastructure and configuration changes through infrastructure-as-code and version-controlled code review, enforcing segregation of duties between change authors and approvers.
- Reviewing employee and system access at least annually, revoking access promptly upon termination, and disabling dormant accounts.
- Assessing third-party vendors for security posture before granting access to production environments or customer data, and maintaining an inventory of vendors classified by data sensitivity and risk level.
- Publishing security commitments and service descriptions externally through the Trust Center, Terms and Conditions, and Privacy Policy so that customers can evaluate the Company's security posture.

> **REVIEW NEEDED**: The commitments above reference specific values (TLS 1.2 minimum, annual penetration testing, HIPAA notification timeframes). Publishing these in the SOC 2 report commits Hatchet to maintaining them throughout future audit periods. Confirm whether to retain the specific figures or adopt more general phrasing where appropriate.

# Components of the System

The system under examination encompasses the infrastructure, software, people, data, and procedures that Hatchet uses to deliver Hatchet Cloud and meet its security commitments. The following sections describe each component and establish the boundaries of the system.

The System is comprised of the following components:
- **Infrastructure** – The collection of physical or virtual resources that supports an overall IT environment, including the physical environment and related structures, IT and hardware (for example, facilities, servers, storage, environmental monitoring equipment, data storage devices and media, mobile devices, and internal networks and connected external telecommunications networks) that Hatchet uses to provide the services.
- **Software** – The application programs and IT system software that supports application programs (operating systems, middleware, and utilities), the types of databases used, the nature of external-facing web applications, and the nature of applications developed in-house, including details about whether the applications in use are mobile applications or desktop or laptop applications.
- **People** – The personnel involved in the governance, operation, security, and use of a system (business unit personnel, developers, operators, user entity personnel, vendor personnel, and managers).
- **Data** – The types of data used by the system, such as transaction streams, files, databases, tables, and output used or processed by the system.
- **Procedures** – The automated and manual procedures related to the services provided, including, as appropriate, procedures by which service activities are initiated, authorized, performed, and delivered, and reports and other information prepared.

## Infrastructure

Hatchet Cloud runs on a multi-cloud infrastructure, with production customer workloads hosted primarily on Amazon Web Services. The Company also leverages Google Cloud Platform for infrastructure services including database, key management, and compute resources; Cloudflare for edge security and traffic management; and additional providers for internal tooling, observability, and network connectivity. All production data processing occurs in data center regions operated by the cloud providers listed below — Hatchet does not operate its own physical data centers.

Employees access production systems through company-managed laptops secured with endpoint protection software, disk encryption, and firewall controls. Remote access to internal and production networks routes through an encrypted mesh VPN.

| Provider | Purpose |
| --- | --- |
| Amazon Web Services (AWS) | Primary cloud hosting for production workloads: compute (EC2), relational databases (RDS), object storage (S3), container registry (ECR), and threat detection (GuardDuty). Data plane regions in the US (Oregon, N. Virginia), EU (Ireland), and Australia (Sydney). |
| Google Cloud Platform (GCP) | Cloud infrastructure including compute instances, Cloud SQL databases, Cloud KMS for cryptographic key management, load balancing, and centralized audit logging. |
| Cloudflare | DNS management, web application firewall (WAF), DDoS protection, TLS termination and enforcement, content delivery, and R2 object storage. |
| ClickHouse | Managed analytics database for customer observability and telemetry data. |
| Ubicloud | Cloud compute for application workloads across US, EU, and Australia regions. |
| Fly.io | Application hosting for internal tooling and services. |
| Vercel | Hosting for the public-facing website, documentation, and frontend assets. |
| Tailscale | Encrypted mesh VPN providing authenticated, end-to-end encrypted network connectivity to internal and production systems. |

> **REVIEW NEEDED**: ClickHouse and Ubicloud appear in the vendor inventory storing customer or confidential data but are not connected as integrations in the compliance platform. Confirm that both are accurately described above — in particular, confirm whether ClickHouse stores customer-facing observability data as part of the delivered platform service, and whether Ubicloud hosts production application workloads or only internal/development environments. If either vendor's role differs, this table and the Subservice Organizations section should be updated accordingly.

## Software

Hatchet's system includes the Company's internally developed platform and third-party applications that support operations, security, and service delivery. The Hatchet platform is a web and API-based application — there are no native mobile applications. Customers interact with the platform through the web console (cloud.hatchet.run), REST and gRPC APIs, and native SDKs for Python, TypeScript, Go, and Ruby.

| Application / Vendor | Purpose |
| --- | --- |
| Hatchet Engine *(in-house)* | Core orchestration engine providing durable execution, task scheduling, workflow management, retry logic, and AI agent coordination. |
| Hatchet API *(in-house)* | External-facing REST and gRPC APIs serving client SDKs, worker connections, and platform management operations. |
| Hatchet Cloud Console *(in-house)* | Web application for workflow monitoring, debugging, configuration, and tenant administration. |
| Hatchet SDKs *(in-house)* | Client libraries enabling developers to define tasks, connect workers, and interact with the platform from application code. |
| GitHub | Version control, pull-request-based code review, CI/CD pipelines, and automated dependency vulnerability scanning (Dependabot). |
| Google Workspace | Identity provider (SSO via Google Sign-In), corporate email, and collaboration. |
| 1Password | Credential and secrets management for team access and secure sharing. |
| Teleport | Privileged access management (PAM) for production infrastructure — short-lived credentials, session recording, and audited access. |
| Slack | Internal communication, access request workflows, and incident coordination. |
| Oneleet | Security compliance platform for control monitoring, evidence collection, policy management, and risk assessment. |
| Oneleet Agent | Endpoint security agent providing MDM capabilities — disk encryption verification, firewall status, screen lock enforcement, and anti-malware monitoring on employee devices. |
| incident.io | Incident management and response coordination. |
| Sentry | Application error tracking and performance monitoring for production services. |
| Betterstack | Uptime monitoring, public status page (status.hatchet.run), and incident alerting. |
| Pganalyze | PostgreSQL database performance monitoring and query analysis. |
| Postmark | Transactional email delivery for customer-facing notifications. |
| GDPR Local | Privacy request management for GDPR data subject access requests. |

## People

Hatchet has nine active personnel: eight employees and one contractor. The Company operates as a small, cross-functional team where most engineers contribute to platform development, infrastructure operations, and security practices.

**Executive leadership and governance.** Alexander Belanger (CEO) and Gabriel Ruttner (CTO) co-founded the Company and form the Founders group. Both executives set security strategy, approve all policies, and make risk tolerance decisions. The two founders jointly review and approve security policies on at least an annual cycle and are both named in the business continuity and disaster recovery escalation chain.

**Security.** Gabriel Ruttner serves as the designated Security Official. In this capacity, the CTO develops and oversees the information security program, manages incident response, conducts risk assessments, approves disposal of classified media, and evaluates control effectiveness. Hatchet does not maintain a separate security team; security responsibilities are embedded in the CTO role and supported by the broader engineering team.

**Engineering and operations.** The engineering team handles platform development, production infrastructure operations, code review, incident response, and day-to-day execution of change management and access control procedures. Access to production systems and customer data is limited to personnel with a demonstrated operational need.

**Contractors.** The Company's contractor is subject to the same confidentiality agreements, policy acknowledgments, training requirements, and access controls as employees. Scope-appropriate adjustments — such as exemption from formal background checks with documented executive approval — are permitted under the Human Resource Security Policy, provided the contractor signs applicable non-disclosure agreements.

**User entity personnel.** Customers interact with the system through the Hatchet Cloud Console, APIs, and SDKs. Customer administrators manage their own tenant configuration, API tokens, and team membership within the platform. Customer personnel do not have access to Hatchet's internal systems or other tenants' data.

## Data

Hatchet classifies all data into four sensitivity levels — **Public**, **Internal**, **Confidential**, and **Secret** — with handling, access, encryption, and disposal requirements escalating at each tier. All customer data is classified as Confidential at minimum, regardless of whether it independently appears sensitive, ensuring uniform protection across the entire customer data set.

The primary data categories the system processes and stores include customer workflow data (definitions, execution parameters, input/output payloads, durable event logs), customer observability data (execution telemetry, monitoring metrics, alerting state, logs), account and tenant metadata (identifiers, emails, authentication tokens, API keys, configuration), and audit and security logs (authentication events, administrative actions, privilege changes, lifecycle events). For customers operating under signed Business Associate Agreements, Hatchet processes protected health information as a Business Associate; PHI is classified as Confidential and subject to the strictest access, disclosure, and disposal controls. Encryption keys, service account credentials, and internal API tokens are classified as Secret.

Hatchet encrypts all data in transit using TLS, enforcing a minimum of TLS 1.2 and preferring TLS 1.3. Cloudflare terminates TLS at the edge with full or strict encryption modes between edge and origin servers. Internal service-to-service traffic transits Tailscale's end-to-end encrypted mesh. All data at rest — production databases, object storage, backups — is encrypted using managed encryption services provided by the underlying cloud infrastructure providers.

Hatchet retains data according to defined schedules tied to data type, classification, and applicable regulation. Customer data retention is governed by the applicable customer contract. Workflow execution logs are retained for up to 90 days. Security logs are retained for a minimum of one year, with PHI-related audit logs retained for at least six years per HIPAA. Account data is retained while active and for up to 12 months after closure. When data reaches the end of its retention period, the Company securely deletes or destroys it using classification-appropriate methods — multi-pass overwrite (NIST 800-88) for Confidential data and physical destruction of storage media for Secret data. Third-party disposal vendors must provide certificates of destruction and maintain chain-of-custody documentation.

## Procedures

### Access Management

Hatchet governs access to systems and data under the principle of least privilege. Every user and service account receives a unique identifier and only the minimum access necessary for its function. The Company prefers SSO via Google Sign-In wherever a service supports it, reducing standalone credential sprawl. Multi-factor authentication is required for all critical services, infrastructure access, and administrative accounts — enforced and monitored across AWS, GCP, Cloudflare, Fly.io, GitHub, Google Workspace, and Slack.

New access follows a formal approval process: requests are submitted in the #service-access-requests Slack channel and require endorsement from at least one member of leadership. Access to critical systems or data additionally requires approval from a security team member (who may be the same person as the manager). Administrative access to production systems — including databases and container orchestration — is granted through Teleport, which issues short-lived credentials rather than standing access. Elevated sessions expire automatically, and all access events are logged and auditable.

The Company reviews access privileges at least annually across all critical vendors and systems. Automated monitors continuously verify that accounts are not dormant, that MFA remains enabled, that offboarded personnel have been fully removed, and that access reviews have been completed within the trailing twelve-month window. When employment or a contractor engagement ends, Hatchet revokes all access within 24 hours using a structured offboarding checklist. Shared accounts are prohibited; service account credentials are rotated whenever a team member with knowledge of the credential departs.

### Change Management

All changes to the Hatchet platform, infrastructure, and configuration follow a standardized change management process. Code and infrastructure-as-code changes are managed through GitHub, where every change requires a pull request with at least one approval from a strongly authenticated reviewer before merging — enforcing segregation of duties between author and approver.

Hatchet tests changes through automated CI pipelines and manual verification before deploying to production. Development, staging, and production environments are maintained as separate environments to prevent unauthorized changes from reaching production. The Company classifies changes by risk and impact; high-risk changes receive additional scrutiny, and authors or reviewers include rollback and test plans where the change's risk warrants them.

A streamlined path exists for emergency changes: implementation may proceed before formal approval, but documentation and approval must be completed within one business day. Changes outside version control — such as manual configuration adjustments to third-party consoles — require explicit affirmative acknowledgment from an approver in an approved communication channel before implementation; silence does not constitute approval.

All changes are logged to maintain an audit trail. Significant changes undergo post-implementation review to verify success and capture lessons learned. The Company communicates relevant system changes to customers through the status page and external release communications, and internally to affected teams.

### Data Communications and Network Security

Hatchet restricts network access to production infrastructure using multiple layers of defense. Cloud security groups and firewall rules block public access to non-essential ports — including SSH and RDP — and prevent database instances from being publicly accessible. Cloud storage buckets are configured to block public access unless a documented business justification is in place, and automated monitors continuously verify these configurations across AWS, GCP, and Cloudflare.

All external traffic to Hatchet Cloud passes through Cloudflare, which provides WAF protection, DDoS mitigation, DNSSEC for supported domains, and automatic HTTPS enforcement. Internal network connectivity between systems uses Tailscale, which authenticates every device and encrypts all traffic end-to-end. Hatchet monitors Tailscale devices for authorization status and flags unauthorized peers for removal. Production database access requires unique authentication credentials, and all database connections are encrypted.

### Logging and Monitoring

Hatchet collects audit logs from all production systems, capturing security-relevant events including login attempts, administrative actions, privilege changes, and major application lifecycle events. The Company transmits logs securely to centralized log management and protects them from tampering. GCP services have data read and write audit logs enabled, GCP subnets have flow logs enabled, and Cloudflare Workers have logging enabled. AWS CloudWatch log groups retain logs for a minimum of 365 days, and GCP log sinks are configured with appropriate retention policies.

Hatchet monitors infrastructure performance across production databases (RDS CPU utilization, I/O activity, free space, freeable memory), load balancers (GCP latency), and network flows. AWS GuardDuty provides continuous threat detection across AWS regions, with findings tracked to remediation. Attack surface scanning runs continuously across company domains. High-risk audit events are reviewed regularly, and any indication of a security incident triggers the incident response process described below.

### Vulnerability Management

Hatchet identifies vulnerabilities through three complementary methods: automated security scanning of production infrastructure (AWS GuardDuty, attack surface scanners), dependency and container image scanning in code and artifact repositories (GitHub Dependabot, AWS ECR image scanning), and at least annual third-party penetration testing performed by an independent firm.

The Company assigns each vulnerability a severity-based remediation target: Critical (P0) within 7 days, High (P1) within 30 days, Medium (P2) within 60 days, and Low (P3) as needed. Automated monitors track remediation progress for ECR image vulnerabilities, GuardDuty findings, attack surface findings, Dependabot alerts, and penetration test findings. Hatchet tests patches in non-production environments before deploying them. For zero-day or actively exploited vulnerabilities, emergency patching follows an accelerated path with testing and approval.

### Endpoint Security

Hatchet secures all employee devices through the Oneleet Agent, which functions as both an MDM solution and an endpoint compliance monitor. The agent continuously verifies that each device maintains disk encryption, an active firewall, screen lock configured to engage after inactivity, and anti-malware protection (Gatekeeper). Automated monitors flag devices that become dormant or fall out of compliance.

Employees connect to internal and production networks through the Tailscale VPN. The Workstation Security Policy prohibits storing PHI or PII on local drives without explicit authorization and encryption, requiring all sensitive data to reside on approved production servers. Lost or stolen devices must be reported immediately and are deactivated upon report.

### Physical Security

Hatchet operates as a distributed team with some employees working from a co-working space and others working remotely. For the co-working environment, the Company relies on the provider's physical perimeter controls — keycards, entry systems, and surveillance — while enforcing additional measures within its own workspace, including clear desk and clear screen policies, visitor escort requirements, and secure storage of sensitive materials in locked cabinets.

Remote employees maintain secure home networks with WPA2 encryption, store devices securely when not in use, and take steps to prevent unauthorized viewing of sensitive information. The Company does not operate its own data centers; physical security of production infrastructure is the responsibility of the cloud service providers described under Infrastructure. Hatchet reviews each provider's physical security documentation annually to confirm it meets the Company's security requirements.

### Incident Response

Hatchet maintains an incident response framework organized into six phases: preparation, identification, containment, eradication, recovery, and lessons learned. The Company classifies incidents by severity (Critical, High, Medium, Low), with defined escalation criteria for each level.

The incident response team uses incident.io to coordinate response activities and Betterstack to communicate status to customers through the public status page. Employees report suspicious activity or potential security incidents immediately. For incidents involving protected health information, Hatchet follows HIPAA-specific breach notification procedures, notifying the affected covered entity within 60 days of discovery and providing the identification of affected individuals, a description of the breach, and steps being taken to investigate and mitigate harm.

After resolution, the Company conducts post-incident reviews to analyze root causes, evaluate response effectiveness, and improve procedures. All incidents and response actions are documented. Hatchet also performs incident response exercises to test capabilities and identify gaps before real incidents occur.

### Backups and Disaster Recovery

Hatchet maintains automated backups for all production systems containing customer data. AWS RDS instances and GCP Cloud SQL databases have automated backups enabled, and Vercel and Cloudflare R2 data is backed up through native platform mechanisms. Nightly full backups of production customer data are retained for a minimum of three days, with longer retention applied where data classification or regulatory obligations require it. Backups are encrypted at rest and in transit, stored in locations geographically separate from the production environment, and restricted to authorized personnel.

The Company tests backup restoration at least annually to validate recovery procedures and documents results. Disaster recovery plans define Recovery Time Objectives and Recovery Point Objectives for each critical system, integrate with the incident response framework, and are reviewed annually or after significant organizational changes.

### Vendor Management

Hatchet maintains an inventory of all third-party vendors, documenting what services each provides, what data each accesses, and the assessed risk level. Before connecting any new technology vendor to the production environment, the Company completes a vendor risk assessment evaluating the vendor's security program, compliance posture (SOC 2, ISO 27001, or equivalent), data segregation controls, and support capabilities for data management and deletion.

The Company classifies vendors by both data sensitivity and risk level. Vendors processing sensitive data — whether internal or on behalf of customers — must demonstrate an established security program with clear documentation. Vendors with potential access to PHI execute Business Associate Agreements that flow down the same obligations Hatchet holds under its own BAAs with covered-entity customers. All vendor agreements establish security expectations and service-level commitments. The vendor inventory and risk assessments are reviewed and updated annually.

### Email Security

Hatchet enforces DMARC across its domains to prevent email spoofing and phishing. Google Workspace email settings block malicious content through protections for anomalous attachment types, attachment scripts, encrypted attachments, linked image malicious content, short URL identification, and untrusted domain warnings. The Company restricts super-administrator mailbox access and does not delegate email account access to non-admin users.

## System Boundaries

The infrastructure, software, people, data, and procedures described in the preceding subsections define the boundaries of the system under examination. The system encompasses all components that deliver Hatchet Cloud, process or store customer data, or support controls that meet the Company's security commitments. Security tools that support controls — including WAF, VPN, endpoint agents, threat detection, vulnerability scanning, and log management — are part of the system and are in scope.

The following systems and services are outside the boundaries of this examination. They do not process or store customer production data, are not necessary for service delivery, and do not support controls relevant to the Company's security commitments:

- Apollo
- Autumn
- Cal.com
- Carta
- Central Business Applications
- Clay
- Common Paper
- Cursor
- Deel
- Discord
- Figma
- Granola
- Lemlist
- Linear
- Mercury
- Notion
- OpenAI
- Anthropic
- PostHog
- Pylon
- Railway
- Stable
- Stripe
- Zoom
- Apple

> **REVIEW NEEDED**: Autumn stores "Payment and Subscription Data" tagged as Customer Data. If Autumn processes customer-identifiable information beyond aggregate billing metadata or gates service access as part of the production control flow, it may need to be brought in scope. Confirm that Autumn handles only billing administration.

> **REVIEW NEEDED**: PostHog collects product usage telemetry and may include customer user identifiers (emails, distinct IDs) depending on configuration. If Hatchet configures PostHog to collect PII from platform end users, it may belong within the system boundary. Confirm whether PostHog ingests identifiable customer data.

> **REVIEW NEEDED**: Railway appears in the vendor inventory with customer and company data. The Company's documentation references Railway as a customer deployment target, not as Hatchet's own infrastructure. Confirm that Railway does not host any of Hatchet's production services or store Hatchet-managed customer data.

Third-party connectivity to the system is managed through authenticated APIs, SSO, VPN, and service-specific access controls. Vendors with access to in-scope systems operate under contractual agreements that define security expectations, and their access is governed by the same least-privilege and monitoring controls described under Procedures. Customer-managed environments — including customer application code, worker runtime environments, and self-hosted Hatchet deployments — are outside the system boundary.

# Control Environment

## Integrity and Ethical Values

Hatchet maintains a Code of Conduct that establishes behavioral expectations for all personnel — employees, contractors, interns, and anyone engaging with the Company in a professional capacity. The Code addresses respect and inclusivity, privacy and security, professionalism, collaboration, and reporting and accountability. All employees formally acknowledge the Code of Conduct, and contractors are required to do the same before beginning work.

The Company supports transparency and accountability through a Whistleblower Policy that enables personnel to report wrongdoing — including harassment, discrimination, policy violations (including HIPAA), and unethical behavior — anonymously and without fear of retaliation. Reports are submitted through a dedicated form and reviewed by the CTO; an alternative escalation path to the CEO exists for complaints involving the CTO.

A formal disciplinary process governs how security violations and policy non-compliance are investigated and resolved. The Security Official investigates reported violations, determines appropriate consequences (up to and including termination), takes steps to prevent recurrence, and retains investigation documentation for a minimum of seven years. Personnel who report violations in good faith are protected from intimidation, threats, or retaliation.

## Management Philosophy and Operating Style

Hatchet's management approach to security reflects the Company's engineering-first culture. Security controls are embedded into development and operations workflows rather than managed as a separate compliance exercise. Infrastructure-as-code, automated CI pipelines, mandatory code review with segregation of duties, and continuous automated compliance monitoring are the primary enforcement mechanisms.

Both co-founders are directly involved in security governance. All security policies are reviewed and approved jointly by the CEO and CTO on at least an annual cycle, and both are named in the business continuity and disaster recovery escalation chain. This direct executive involvement ensures that security decisions are made with full business context and that risk tolerance is set by the individuals most accountable for the Company's operations.

## Organizational Structure and Assignment of Authority

Hatchet's organizational structure is flat, consistent with its size and stage. Authority and accountability are clearly assigned:

- **Alexander Belanger (CEO)** sets Company strategy, approves security policies, and serves as the final escalation point for security violations and privacy matters.
- **Gabriel Ruttner (CTO and Security Official)** owns the information security program — developing and overseeing policies and procedures, managing incident response, conducting risk assessments, approving disposal of classified media, and evaluating control effectiveness.
- **The engineering team** is responsible for platform development, infrastructure operations, code review, and day-to-day execution of security procedures including change management, access control, and incident response.

Roles and responsibilities are defined in the information security program and communicated to all personnel during onboarding. The assignment of security responsibilities is reinforced through policy acknowledgment, training completion tracking, and access controls that align system permissions with job function.

## Commitment to Competence

Hatchet evaluates competence during hiring and maintains it through ongoing training and performance management. The Company conducts background checks or reference checks on prospective employees, contractors, and third parties who will access sensitive data or production systems, scaled to the role's level of access and risk exposure.

All personnel complete general security awareness training within 30 days of onboarding and annually thereafter. The training covers social engineering identification and response, workstation security, common attack vectors, and sensitive data handling. In addition, all personnel complete HIPAA training (covering PHI types, obligations, and penalties) and GDPR training (covering personal data types, core principles, and data subject rights), reflecting the Company's regulatory obligations. Training completion is tracked through automated monitors that flag overdue personnel.

Hatchet conducts regular performance evaluations that integrate security practices — recognizing good security behavior, constructively addressing security mistakes, and assessing security awareness for roles with significant system or data access.

## HR Policies and Practices

Hatchet maintains a Human Resource Security Policy governing the security dimensions of the entire employment lifecycle.

**Hiring.** Prospective personnel undergo background and/or reference checks appropriate to their role before being granted access to sensitive data or production systems. All new personnel sign non-disclosure and confidentiality agreements before accessing Company data or systems. Contractors with limited scope and no exposure to sensitive data may be exempted from formal background checks with documented executive approval, but must still sign confidentiality agreements before any access is granted.

**Onboarding.** The Company follows a structured onboarding process tracked through a dedicated checklist. New personnel are provisioned with role-appropriate access, complete required training (general security, HIPAA, GDPR), and acknowledge all applicable policies. Automated monitors verify that every new hire completes the onboarding checklist.

**During employment.** All personnel formally acknowledge and sign applicable security policies, tracked by automated monitors. Role-based and compliance-specific training is assigned based on responsibilities and regulatory requirements. Policy compliance is mandatory, and violations trigger the formal disciplinary process.

**Offboarding.** A structured offboarding checklist governs all departures. Hatchet revokes all system access within 24 hours or one business day, collects company property, and reminds departing personnel of ongoing confidentiality obligations that extend beyond the end of the relationship. Automated monitors verify that every offboarded individual has been fully removed from integrated systems.

# Risk Assessment

Hatchet conducts formal risk assessments at least annually to identify and evaluate threats and vulnerabilities that could affect the Company's ability to meet its security commitments. The Company's Risk Management Policy defines the methodology: each identified risk is evaluated on likelihood and potential impact (financial, operational, reputational, regulatory) and rated as high, medium, or low.

High residual risks — such as regulatory non-compliance or significant data exposure — receive the highest priority treatment with immediate corrective actions. Medium risks are actively managed with documented contingency plans. Low risks are monitored to prevent escalation. Any risk exceeding defined tolerance levels is escalated to senior management for review.

For each risk, the Company assigns a risk owner and records a treatment plan in a risk register specifying the chosen strategy (mitigate, avoid, transfer, or accept), responsible parties, required resources, and deadlines. Fraud risk is explicitly identified and assessed as part of the process. Risk monitoring occurs continuously, with formal reviews of control effectiveness on an annual basis.

**Vendor and third-party risk.** Hatchet addresses risks from vendors and third parties through its vendor management program. Before connecting any new technology vendor to the production environment, the Company completes a vendor risk assessment evaluating the vendor's security program, compliance certifications, data segregation controls, and support for data management and deletion. Vendors are classified by data sensitivity and risk level. Those handling sensitive or customer data must demonstrate an established security program, and vendors with potential access to PHI execute Business Associate Agreements that extend the same protective obligations. The vendor inventory and associated risk assessments are reviewed annually.

# Information and Communication

**Internal communication.** Hatchet communicates security policies, responsibilities, and control requirements to personnel through multiple reinforcing channels. All security policies are maintained in the compliance platform and require formal acknowledgment by relevant personnel — tracked by automated monitors that flag unsigned policies. New personnel receive policies and training during the onboarding process. The Company uses Slack for day-to-day communication of security-relevant matters, including access requests, incident coordination, and change notifications. Findings from risk assessments, control monitoring, and internal audits are escalated to management through defined reporting lines. When policies are updated, the Company communicates changes through internal channels and requires re-acknowledgment.

**External communication.** Hatchet communicates with customers and stakeholders through several defined mechanisms. The Trust Center (trust.hatchet.run) publishes the Company's compliance posture, active frameworks, and security documentation. The status page (status.hatchet.run) provides real-time availability information and incident history for Hatchet Cloud, including API availability and the web console. System changes and maintenance activities that affect customers are communicated through the status page and external notifications. The Terms and Conditions, Privacy Policy, and DPA are published on hatchet.run and define the contractual framework under which the service operates. For privacy inquiries and data subject access requests, the Company provides a contact channel at legal@hatchet.run.

# Monitoring

**Ongoing automated monitoring.** Hatchet monitors control effectiveness on a continuous basis through automated monitors integrated into its compliance platform and cloud infrastructure. The Company maintains automated checks across its integrated systems that alert when controls drift from their expected state. These monitors cover access control (dormant account detection, MFA enforcement verification, access review completion, offboarding verification across Fly.io, Tailscale, Google Workspace, and other systems), infrastructure security (public bucket exposure on AWS and GCP, firewall and security group configurations, public SSH/RDP access, WAF status), endpoint compliance (disk encryption, firewall, screen lock, anti-malware status via Oneleet Agent), vulnerability management (ECR image scan findings, GuardDuty alerts, attack surface findings, Dependabot alerts, penetration test remediation), change management (code review approval requirements, deployment access restrictions), logging (CloudWatch retention, GCP audit log enablement, GCP flow logs, Cloudflare worker logging), data protection (encryption at rest on Cloud SQL, R2, Vercel; encryption in transit via TLS enforcement on Cloudflare zones), and email security (DMARC enablement, Google Workspace malicious content protections).

Infrastructure performance monitoring tracks production database health (RDS CPU utilization, I/O, free space, freeable memory), load balancer latency, and network flow data. AWS GuardDuty provides continuous threat detection, with all findings tracked through to remediation.

**Periodic evaluations.** Hatchet performs an internal security audit annually to evaluate control effectiveness, verify compliance, and identify improvement areas. Risk assessments are conducted annually, and the risk register is updated for any material changes between cycles. Employee access reviews are completed at least annually across all critical systems and vendors. All security policies are reviewed and re-approved on an annual cycle by executive leadership.

**Subservice organization monitoring.** Hatchet monitors the services performed by its subservice organizations through both automated and manual activities. The compliance platform maintains connected integrations with AWS, GCP, and Cloudflare that continuously monitor configuration state, security posture, and control adherence through the automated monitors described above. The Company reviews independent third-party assessment reports (such as SOC 2 reports) from its subservice organizations annually and monitors their public status pages for service disruptions or material changes.

**Deficiency remediation.** When monitoring activities, reviews, or assessments identify a control deficiency — through automated alerts, audit findings, or management review — Hatchet logs the finding, assigns an owner, and tracks it to remediation. Vulnerability remediation follows severity-based timelines (Critical within 7 days, High within 30 days, Medium within 60 days). Penetration test findings are individually tracked and verified as resolved. Unresolved items are escalated to management for prioritization.

# Subservice Organizations

The description does not extend to the services provided by Amazon Web Services, Google Cloud Platform, Cloudflare, and ClickHouse (collectively, the subservice organizations). Section 4 of this report and the description of the system only cover the relevant trust services criteria and related controls in support of the achievement of Hatchet's service commitments and system requirements and exclude the related controls of the subservice organizations.

Although the subservice organizations have been carved out for the purposes of this report, Hatchet's management has assumed, in the design of the system, that certain complementary subservice organization controls (CSOCs) would be implemented by the subservice organizations. Such controls are necessary, in combination with controls at Hatchet, to provide reasonable assurance that Hatchet's service commitments and system requirements were achieved. Because the related service commitments and system requirements can only be achieved if the CSOCs are suitably designed and operating effectively during the audit period, each user entity must evaluate Hatchet's controls, related tests of controls, and results of tests described in Section 4 of this report, considering the types of related CSOCs expected to be implemented at the subservice organizations as shown below.

The subservice organizations and their roles are:

- **Amazon Web Services** — primary cloud hosting for production infrastructure, described under Infrastructure.
- **Google Cloud Platform** — cloud infrastructure including compute, databases, KMS, and logging, described under Infrastructure.
- **Cloudflare** — edge security, DNS, WAF, TLS termination, and content delivery, described under Infrastructure.
- **ClickHouse** — managed analytics database for customer observability data, described under Infrastructure.

The following table maps the complementary subservice organization controls that each subservice organization is expected to implement. These mappings are based on the nature of services each provider delivers and should be verified against each subservice organization's SOC 2 report or equivalent independent assessment.

| Subservice Organization | Complementary Subservice Organization Control |
| --- | --- |
| Amazon Web Services | Restrict physical access to data center facilities to authorized personnel. |
| Amazon Web Services | Implement environmental protections — fire detection and suppression, climate control, and power redundancy — to safeguard infrastructure from environmental threats. |
| Amazon Web Services | Securely dispose of or sanitize storage media containing customer data before reuse or decommissioning. |
| Amazon Web Services | Restrict logical access to production host infrastructure, hypervisors, and management planes to authorized personnel. |
| Amazon Web Services | Monitor cloud infrastructure for security events, anomalies, and unauthorized access attempts. |
| Amazon Web Services | Encrypt data at rest on managed storage and database services using industry-standard encryption algorithms. |
| Amazon Web Services | Detect and mitigate network-level and application-level attacks against cloud infrastructure. |
| Google Cloud Platform | Restrict physical access to data center facilities to authorized personnel. |
| Google Cloud Platform | Implement environmental protections to safeguard infrastructure from environmental threats. |
| Google Cloud Platform | Securely dispose of or sanitize storage media before reuse or decommissioning. |
| Google Cloud Platform | Restrict logical access to Cloud KMS key material and cryptographic operations to authorized services and personnel. |
| Google Cloud Platform | Monitor cloud infrastructure for security events, anomalies, and unauthorized access attempts. |
| Google Cloud Platform | Encrypt data at rest on managed storage and database services using industry-standard encryption algorithms. |
| Cloudflare | Maintain WAF rule sets and DDoS mitigation controls to protect customer traffic traversing the edge network. |
| Cloudflare | Enforce TLS termination and encrypt data in transit across edge infrastructure using current protocol versions. |
| Cloudflare | Restrict physical and logical access to network edge infrastructure to authorized personnel. |
| Cloudflare | Monitor edge network infrastructure for security events, anomalies, and unauthorized access attempts. |
| ClickHouse | Restrict physical and logical access to managed database infrastructure to authorized personnel. |
| ClickHouse | Encrypt customer data at rest on managed storage using industry-standard encryption algorithms. |
| ClickHouse | Monitor database infrastructure for security events and unauthorized access attempts. |
| ClickHouse | Maintain backup and recovery controls for customer data stored in managed databases. |

Management of Hatchet receives and reviews independent third-party assessment reports of its subservice organizations annually. In addition, management monitors the services performed by the subservice organizations through connected integrations in the compliance platform that continuously assess configuration state and security posture for AWS, GCP, and Cloudflare — including automated checks on encryption settings, access controls, logging enablement, and network exposure. Management monitors the subservice organizations' status pages to stay informed of service disruptions or material changes and maintains customer support channels to relay issues or concerns to subservice organization management.

> **REVIEW NEEDED — Subservice Organization Classification and Monitoring Activities:**
>
> **Included as subservice organizations (confirm each):**
>
> - **Amazon Web Services** — Hosts production infrastructure and stores customer data (classified Secret, tagged Customer Data / PHI / PII). Hatchet cannot perform physical security, media disposal, or host-level infrastructure controls at AWS data centers; these must be performed by AWS. AWS is **not** marked [Subprocessor] on the trust center — confirm this is an oversight and update the subprocessor list accordingly.
> - **Google Cloud Platform** — Extensive control monitoring (Cloud SQL databases, compute instances, KMS keys, load balancers, logging sinks) indicates GCP runs production infrastructure components. The data inventory classifies GCP-stored data as Secret ("Credentials and Secrets"). If GCP manages encryption keys (via Cloud KMS) that protect customer data, or hosts production databases or compute in the service delivery path, its physical and logical infrastructure controls are necessary for Hatchet's security commitments. GCP is **not** marked [Subprocessor]. Confirm whether GCP hosts production components in the service delivery path, or only internal operational infrastructure — if the latter, reclassify to bucket 2 and remove from this section.
> - **Cloudflare** — All external customer traffic routes through Cloudflare for WAF, DDoS protection, and TLS termination. Hatchet's automated monitors track Cloudflare TLS configuration (TLS 1.3, minimum 1.2, full/strict encryption mode), WAF status, DNSSEC, and proxy enablement — indicating the system design depends on Cloudflare to fulfill encryption-in-transit and network protection commitments. Cloudflare is **not** marked [Subprocessor]; confirm and update the subprocessor list.
> - **ClickHouse** — Stores customer observability data (classified Confidential, tagged Customer Data) that supports the platform's monitoring and logging features. No evidence of client-side encryption — ClickHouse can read this data. If ClickHouse's data protection controls fail, customer data confidentiality is affected. ClickHouse is **not** marked [Subprocessor] and has no connected integration in the compliance platform (no automated monitoring). Confirm ClickHouse is in the service delivery path, add it to the subprocessor list, and confirm whether Hatchet currently reviews a ClickHouse SOC 2 or equivalent report.
>
> **Considered but excluded (bucket 2 — third party with system access, not subservice organizations):**
>
> - **Teleport** [Subprocessor] — PAM tool for internal production access (session recording, short-lived credentials). Not in the service delivery path: customers do not interact with Teleport, and Hatchet's access control commitments are achievable through its own controls. The [Subprocessor] marker indicates Teleport processes data as a subprocessor under privacy law, but the subservice organization test is narrower. Confirm whether any security commitment depends on a control Teleport performs (e.g., integrity of session recordings for audit). If so, reclassify as a subservice organization.
> - **Postmark** — Delivers transactional customer emails (tagged Customer Data). If email notifications are a feature of the platform rather than an operational convenience, and Postmark can read message content that constitutes customer data, Postmark may belong in bucket 1. Confirm whether platform alerting/notification features depend on Postmark delivery and whether the email content includes customer production data.
> - **Ubicloud** — Stores Confidential "Application Data" in US, EU, and Australia, with processing locations matching Hatchet Cloud regions. Not connected as an integration. If Ubicloud hosts production workloads serving customers, it is likely a subservice organization. Confirm Ubicloud's role and reclassify accordingly.
> - **Vercel** — Hosts the public website and frontend assets. Data inventory shows only Public-classified website traffic. Not in the customer data delivery path for the orchestration service.
>
> **Monitoring activities:** The narrative above asserts that Hatchet reviews subservice organization SOC 2 reports annually, monitors status pages, and uses connected integrations for continuous posture assessment. Confirm these statements match actual practice — particularly whether Hatchet currently holds and reviews SOC 2 (or equivalent) reports from all four named subservice organizations, and whether status page monitoring is formalized or ad hoc.

# System Incidents

## A. Incidents Pertaining to the Audit Period

> **INPUT NEEDED**: Confirm whether any significant system incidents occurred during the audit period (including incidents that occurred during the period and were subsequently identified after period end but before report issuance) that resulted in a failure to achieve Hatchet's service commitments and system requirements. Also consider whether any known incidents at subservice organizations should be disclosed. If any occurred, describe their nature, timing, effect, and remediation.
>
> If none occurred, you can use the following:
>
> "No significant security incidents have occurred during the audit period that would have a material effect on the suitability of the design and operating effectiveness of the controls and/or description of the system covered in Section 3 of this report."

## B. Incidents Subsequent to the Audit Period

> **INPUT NEEDED**: Confirm whether any significant system incidents occurred subsequent to the audit period but before report issuance. If any occurred, describe their nature, timing, effect, and remediation.
>
> If none occurred, you can use the following:
>
> "Management is not aware of any incidents that occurred subsequent to the period covered by Section 3 of this report through the date of the service auditor's report that would have a significant effect on management's assertion and description of its system."

# Significant Changes During the Audit Period

## A. Significant Changes and Events During the Audit Period

> **INPUT NEEDED**: Confirm whether any significant changes or events occurred during the audit period (including changes that occurred during the period and were subsequently identified after period end but before report issuance). Significant changes include:
> - Changes to the services provided
> - Significant changes to IT and security personnel
> - Significant changes to system processes, IT architecture, or applications
> - Changes to legal and regulatory requirements affecting system requirements
> - Changes to organizational structure resulting in a change to internal control
>
> If any occurred, describe them here. If none occurred, you can use the following:
>
> "No significant changes and events have occurred during the audit period that would have a material effect on the suitability of the design and operating effectiveness of the controls and/or description of the system covered in Section 3 of this report."

## B. Significant Changes and Events Subsequent to the Audit Period

> **INPUT NEEDED**: Confirm whether any significant changes or events occurred subsequent to the audit period but before report issuance. If any occurred, describe them here.
>
> If none occurred, you can use the following:
>
> "Management is not aware of any changes that occurred subsequent to the period covered by Section 3 of this report through the date of the service auditor's report that would have a significant effect on management's assertion and description of its system."

# Applicable Trust Services Criteria

All Security criteria are applicable to the system. The Availability, Processing Integrity, Confidentiality, and Privacy categories are not included within the scope of this examination.

# Complementary User Entity Controls

Hatchet's services are built on a shared responsibility model. While Hatchet implements comprehensive controls on its side, certain security measures must be implemented by its customers in their own environments to ensure the overall security of the system. In the context of this report, these are referred to as "complementary user entity controls."

Not all Trust Services Criteria can be met by Hatchet's controls alone. Customers should implement the following controls to provide additional assurance that the criteria described within this report are met. These represent key controls but are not exhaustive — auditors reviewing customer compliance should exercise judgment in determining what additional controls may apply.

| Trust Services Criteria | Complementary User Entity Controls |
|------------------------|-----------------------------------|
| CC2.1 | User entities are responsible for the security and integrity of their own data, including data used by Hatchet systems and services. |
| CC6.2 | Authorized personnel at the user entity determine which individuals need access to Hatchet and grant permissions accordingly, including managing any API keys or credentials used to access the platform. |
| CC6.3 | User entities periodically review authorized users and their access privileges. |
| CC6.6 | User entities are responsible for protecting their data in transit to Hatchet. |
| CC6.6 | User entities should maintain physical security and environmental controls for all devices and access points at their facilities — including remote employees — that connect to Hatchet. |
