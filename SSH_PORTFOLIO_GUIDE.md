# Building an SSH Portfolio (and Hosting It for Real)

If you’ve seen people publish portfolios you “visit” over SSH, you’ve probably wondered two things:
1) how that’s possible, and
2) how much infrastructure it actually takes.

Here’s the full story of building one myself—what SSH is, why an SSH portfolio is fun, how Charm makes it easy, and how to host it properly without breaking your admin access.

---

## What is SSH (quickly)?

SSH (Secure Shell) is a protocol that lets you securely connect to another machine over the internet and get a terminal session. Normally you use it to manage servers:

```
ssh user@server.com
```

But SSH doesn’t have to be just for sysadmin work. It can be a public-facing interface, too.

---

## What’s an SSH portfolio?

It’s a terminal UI that people can SSH into and explore—like a live résumé inside their terminal.

If you’ve seen ThePrimeagen’s `terminal.shop`, that’s a great example of the vibe: it turns SSH from “server admin” into “interactive experience.”

---

## Why Charm makes this easy

Charm’s stack (Wish + Bubble Tea) makes SSH apps feel like regular TUI projects.

- **Wish**: runs the SSH server
- **Bubble Tea**: handles the interactive UI

That means you can build a fun TUI locally, and then let Wish serve it over SSH with just a few lines of setup.

At a high level your server setup looks like this:

```go
wish.NewServer(
    wish.WithAddress("0.0.0.0:22"),
    wish.WithHostKeyPath(".ssh/host_ed25519"),
    wish.WithPublicKeyAuth(publicKeyAuth),
    wish.WithPasswordAuth(passwordAuth), // optional fallback
    wish.WithMiddleware(bubbletea.Middleware(teaHandler)),
)
```

---

## Hosting options (Azure, EC2, Fly.io)

You can host this anywhere that gives you a public IP and lets you control port 22:

- **Azure VMs**
- **AWS EC2**
- **Fly.io**

I went with **EC2** because I had AWS credits and didn’t want to fight with custom ports on a PaaS.

---

## The port 22 problem (and why it matters)

Port **22** is the default SSH port. If you want people to simply run:

```
ssh yourdomain.com
```

your app must listen on **port 22**.

But there’s a catch:

- **Your admin SSH** also runs on port 22 by default.
- You can’t run two SSH servers on the same port.

So the fix is:

1) move admin SSH to a different port (like 22022), and  
2) put the portfolio app on port 22.

---

## How we moved admin SSH

On the VM:

1. Edit `/etc/ssh/sshd_config`
2. Add:
   ```
   Port 22022
   ```
3. Restart SSH:
   ```
   sudo systemctl restart sshd
   ```

Then update the cloud firewall (AWS Security Group):

- TCP **22022** from your own IP (admin)
- TCP **22** from the world (portfolio)

---

## Running the app on port 22

Binding to port 22 requires elevated permission, so you give the binary the right capability:

```
sudo setcap 'cap_net_bind_service=+ep' /home/ec2-user/joe-ssh-linux
```

Then run it with systemd so it stays up:

```
sudo systemctl enable --now joe-ssh
```

---

## Adding a domain

Once the app is live on port 22, add an `A` record pointing your domain to the VM’s public IP. For example:

- `ssh.yourdomain.com -> YOUR_IP`
- `joesluis.dev -> YOUR_IP`

If you use Cloudflare, make sure the record is **DNS only** (not proxied), since SSH can’t go through their proxy.

---

## Final test

From any machine:

```
ssh yourdomain.com
```

And you should see your terminal portfolio.

