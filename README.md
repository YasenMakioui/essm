# essm

```
 ███████╗███████╗███████╗███╗   ███╗
 ██╔════╝██╔════╝██╔════╝████╗ ████║
 █████╗  ███████╗███████╗██╔████╔██║
 ██╔══╝  ╚════██║╚════██║██║╚██╔╝██║
 ███████╗███████║███████║██║ ╚═╝ ██║
 ╚══════╝╚══════╝╚══════╝╚═╝     ╚═╝
        e ▸ ssm  ·  ec2 over ssm
```

**A tiny terminal UI for finding EC2 instances and connecting to them over SSM — without memorizing instance IDs or name tags.**

Connecting to an instance with Session Manager normally means knowing the exact instance ID or `Name` tag up front — so you end up listing instances or digging through the console first. `essm` turns that into a browsable list: see your instances, see at a glance which ones are reachable over SSM, peek at an instance's details, and press <kbd>Enter</kbd> to start a session.

> **Scope by design:** `essm` is a **visualization + connection** tool only. It never starts, stops, terminates, or modifies instances. It reads, and it connects. Nothing else.

---

## Features

- **Instance list** with live **SSM reachability** — green `● online` for instances you can actually reach, grey `○ offline` for the rest.
- **Fast search** by instance ID or `Name` tag (`/`).
- **Detail panel** (`i`) — loaded on demand so the list stays fast: type, AMI, private/public IP, subnet, VPC, IAM instance profile, security groups, attached volumes, and tags.
- **One-key connect** (`Enter`) — hands off to `aws ssm start-session`.
- **Auto-refresh** every 30s, plus manual refresh (`r`).

## Demo

![essm demo](demo.gif)

---

## Requirements

Runtime:

- [**AWS CLI v2**](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) — `essm` shells out to `aws ssm start-session`.
- [**Session Manager plugin**](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) — required by `start-session`.
- **AWS credentials** configured in your environment (env vars, a shared profile, or SSO). `essm` uses the default AWS SDK credential chain.
- For an instance to appear as `● online`, it must have the **SSM agent** running and an instance role granting `AmazonSSMManagedInstanceCore` (standard SSM setup).

Build:

- **Go 1.25+**

Built on [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss) (charm.land v2), plus the [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2).

---

## Install

### Download a release

Grab a prebuilt binary for your platform from the [**releases page**](https://github.com/YasenMakioui/essm/releases), or install with Go:

```bash
go install github.com/YasenMakioui/essm@latest
```

### Build from source

```bash
git clone https://github.com/YasenMakioui/essm.git
cd essm
go build -o essm .
./essm
```

---

## Usage

1. Authenticate with AWS — for example:

   ```bash
   aws sso login
   ```

   (or configure a profile / credentials however you normally do; set `AWS_PROFILE` / `AWS_REGION` as needed).

2. Run it:

   ```bash
   ./essm
   ```

3. Navigate to an instance and press <kbd>Enter</kbd> to open a Session Manager session. When you exit the session, you're back in `essm`.

### Keybindings

| Key | Action |
| --- | --- |
| <kbd>↑</kbd> / <kbd>↓</kbd> (<kbd>k</kbd> / <kbd>j</kbd>) | Move selection |
| <kbd>PgUp</kbd> / <kbd>PgDn</kbd> | Page through the list |
| <kbd>/</kbd> | Search by instance ID or Name |
| <kbd>i</kbd> | Toggle the detail panel for the selected instance (updates as you navigate) |
| <kbd>r</kbd> | Refresh the instance list |
| <kbd>Enter</kbd> | Connect to the selected instance via SSM |
| <kbd>Esc</kbd> | Exit search / close the detail panel |
| <kbd>q</kbd> / <kbd>Ctrl</kbd>+<kbd>C</kbd> | Quit |

---

## IAM permissions

`essm` needs read access to describe instances and their SSM status, and permission to start sessions:

| Action | Used for |
| --- | --- |
| `ec2:DescribeInstances` | Listing instances and loading the detail panel |
| `ssm:DescribeInstanceInformation` | Determining SSM reachability (`● online` / `○ offline`) |
| `ssm:StartSession` | Connecting (via `aws ssm start-session`) |
| `ssm:TerminateSession` | Cleanly ending a session |

A minimal policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "essmRead",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ssm:DescribeInstanceInformation"
      ],
      "Resource": "*"
    },
    {
      "Sid": "essmConnect",
      "Effect": "Allow",
      "Action": [
        "ssm:StartSession",
        "ssm:TerminateSession"
      ],
      "Resource": "*"
    }
  ]
}
```

> Scope `Resource` down to the instances/accounts you care about for production use. `ssm:StartSession` can be restricted to specific instances or tags.

---

## How it works

- The list joins two API calls: `ec2:DescribeInstances` (what instances exist) and `ssm:DescribeInstanceInformation` (which are registered and `Online` in SSM), matched by instance ID.
- The detail panel calls `ec2:DescribeInstances` for a **single** instance, and only when you press <kbd>i</kbd> — so the main list never pays for detail it isn't showing.
- Connecting simply execs `aws ssm start-session --target <instance-id>`; `essm` suspends while the session runs and resumes when it ends.

---

## Roadmap

- [ ] Resolve the true IAM **role** behind the instance profile (extra `iam:GetInstanceProfile` call) — currently shows the instance-profile name
- [ ] Richer volume info (size / type via `ec2:DescribeVolumes`)

---

## License

Released under the [MIT License](LICENSE).
