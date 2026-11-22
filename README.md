[portfolio_readme_for_legaltech_institutions (1).md](https://github.com/user-attachments/files/35228481/portfolio_readme_for_legaltech_institutions.1.md)
# Singapore Playbook Contract Reviewer

A local-first contract auditing and redlining tool built in Go and powered by local LLMs via Ollama. Built specifically for Singapore residential tenancy agreements, it evaluates contracts against structured legal playbooks, verifies every quoted clause directly against the source text, and drafts compliant fallback clauses—all without client data ever leaving your computer.

> Disclaimer: This software is a LegalTech portfolio demonstration and does not constitute legal advice. Statutory citations, thresholds, and fallback clauses are provided for illustrative purposes only. A Singapore-qualified Advocate & Solicitor should review all playbook rules before reliance in live transactions.

## Why I Built This

In routine legal and property work, reviewing standardized contracts against internal checklists takes significant time. While cloud AI tools help speed things up, using them in legal environments raises three practical concerns:

1. Client Confidentiality and PDPA Risk: Uploading unredacted agreements containing NRIC/FIN numbers, addresses, and financial terms to external cloud APIs creates data residency and Singapore Personal Data Protection Act (PDPA) exposure.
2. Inconsistent Review Standards: Open-ended prompts ("Review this contract for risks") produce unpredictable results that drift from firm-approved risk tolerances.
3. Quote Hallucination: Generative models often alter or invent contractual wording when summarizing issues, making it harder for reviewers to trust the output.

SG-Playbook-Reviewer solves these issues by keeping inference 100% local and separating legal rules (structured JSON playbooks) from language model execution (schema-constrained local inference).

## How It Works

### 1. Rules Stored as Data (Playbook Engine)
Legal review standards are stored as editable JSON rules (`internal/playbook/`) rather than buried inside prompt strings. Each rule defines:

- Rule ID and Title (e.g., `SG-TA-03: Prohibition on Self-Help Lockout and Seizure`)
- Severity Tier (High, Medium, Low)
- Statutory or Regulatory Reference (e.g., Distress Act 1934, Stamp Duties Act 1929, URA Planning Act 1998)
- Evaluation Criteria and Mandatory Clause Flags

Reviewers can toggle rules on or off or adjust commercial thresholds (such as minor repair caps or diplomatic clause triggers) directly in the app without touching code.

### 2. Two-Pass Audit and Redline Pipeline
To keep responses accurate on lightweight local models (`qwen3.5:4b`), the engine splits the review into two structured steps via Ollama's JSON schema API:

- Pass 1 (Compliance Audit): Checks the contract strictly against active rules, marking each rule as `PASS`, `FLAGGED`, or `MISSING_CLAUSE` along with a plain-English risk explanation and exact clause excerpt.
- Pass 2 (Fallback Drafting): Runs only on `FLAGGED` and `MISSING_CLAUSE` items to draft balanced, Singapore-law-aligned replacement clauses and negotiation notes.

### 3. Exact-Quote Verification
Every quoted snippet returned by the model in Pass 1 is checked word-for-word against the original contract text in Go before being displayed. Unverified quotes are flagged immediately, ensuring the Locate in Contract button jumps straight to the exact matching sentence.

### 4. Zero-Egress Windows Desktop App
Compiled as a standalone Windows GUI executable (`-ldflags="-H=windowsgui"`) with an embedded frontend (`go:embed`), the app communicates only over localhost (`127.0.0.1`) with Ollama. There are no external network calls, telemetry, or cloud dependencies.

## Singapore Residential Tenancy Playbook Coverage

The default embedded playbook (`default_sg_tenancy.json`) covers eight core statutory and market-standard checks under Singapore law:

| Rule ID | Legal Domain | Governing Statute / Guideline | What It Checks |
| :--- | :--- | :--- | :--- |
| `SG-TA-01` | Stamp Duty and Stamping Window | Stamp Duties Act 1929 (IRAS) | Checks stamp duty allocation and compliance with the 14-day stamping deadline (or 30 days if signed overseas). |
| `SG-TA-02` | Minimum Stay and Occupancy Caps | Planning Act 1998 (URA Guidelines) | Flags fixed terms below statutory minimums (3 months for private residential; 6 months for HDB) and excessive unrelated occupancy caps. |
| `SG-TA-03` | Self-Help Lockout and Seizure | Distress Act 1934 | Flags unilateral landlord re-entry, lockouts, utility disconnections, or seizure/sale of tenant goods without a court writ. |
| `SG-TA-04` | Security Deposit and Wear and Tear | CEA Residential Tenancy Guidelines | Checks deposit proportionality (1 month per year of lease), timely refund windows (14 days), and fair wear-and-tear exclusions. |
| `SG-TA-05` | Minor Repair Liability Cap | CEA Standard Tenancy Template | Flags unlimited tenant repair liability or shifting of structural/major appliance replacement onto the tenant (standard cap: S$150–S$200). |
| `SG-TA-06` | Diplomatic Clause | Singapore Expatriate Lease Standard | Ensures leases of 24 months or longer include an early termination right after 12 months (with 2 months' notice) upon relocation or employment cessation. |
| `SG-TA-07` | Personal Data Governance | Personal Data Protection Act 2012 (PDPA) | Validates that NRIC/FIN and occupant personal data collection is limited to tenancy administration and statutory reporting. |
| `SG-TA-08` | Governing Law and Dispute Forum | Small Claims Tribunals Act 1984 | Flags foreign governing law or mandatory offshore arbitration clauses that oust Singapore courts and Small Claims Tribunals jurisdiction. |

## Repository Structure

```text
├── cmd/
│   └── desktop/
│       └── main.go                  # Windows GUI entry point, embedded server & lifecycle manager
├── internal/
│   ├── exporter/
│   │   └── markdown.go              # Formats audit findings and redlines into Markdown reports
│   ├── llm/
│   │   └── ollama.go                # Local Ollama HTTP client with JSON Schema enforcement
│   ├── playbook/
│   │   ├── default_sg_tenancy.json  # Machine-readable Singapore tenancy ruleset
│   │   ├── loader.go                # Embedded & custom JSON playbook loader
│   │   └── models.go                # Data structures for rules, findings, and redlines
│   └── reviewer/
│       └── engine.go                # Two-pass orchestration engine & exact-substring verifier
├── web/
│   ├── embed.go                     # Compile-time asset embedding (go:embed)
│   └── index.html                   # Split-pane contract viewer, playbook editor & redline inspector
├── go.mod                           # Zero-external-dependency Go module definition
└── README.md
```

## Application Preview

Add a screenshot of `SG-Playbook-Reviewer.exe` analyzing a non-compliant tenancy agreement here (you can drag and drop an image directly into this section in GitHub's editor).

## Getting Started (Windows)

### Prerequisites

- Go 1.22 or higher
- Ollama installed locally with a lightweight model pulled (recommended for laptop memory efficiency):

```powershell
ollama pull qwen3.5:4b
```

### Build and Run

1. Clone the repository:

```powershell
git clone https://github.com/waydenoh/LegalTech-Project-1-2026.git
cd LegalTech-Project-1-2026
```

2. Build the standalone Windows desktop executable (no console window):

```powershell
go build -ldflags="-H=windowsgui -s -w" -o SG-Playbook-Reviewer.exe ./cmd/desktop
```

3. Launch the app:
Double-click `SG-Playbook-Reviewer.exe` (or run `.\SG-Playbook-Reviewer.exe`). Click Sample (Traps) in the top bar to run an immediate test review and export a Markdown redline report.

## Planned Improvements

- Additional Playbooks: Support for Singapore Commercial Leases, Mutual NDAs, and Employment Contracts under the Employment Act 1968.
- Document File Import: Local text extraction for `.docx` and `.pdf` agreements while preserving clause numbering.
- Tracked-Changes Export: Direct export of `.docx` redlines using OpenXML revision tags for Microsoft Word workflows.
