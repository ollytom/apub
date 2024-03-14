# Apas: ActivityPub via email

By Oliver Lowe <[o@olowe.co](mailto:o@olowe.co)> ([otl@apubtest2.srcbeat.com](https://apas.srcbeat.com/otl/actor.json))

> Every program attempts to expand until it can read mail.
> Those programs which cannot so expand are replaced by ones which can.

—[Zawinski's Law of Software Envelopment]

**[Source code]** | **[GoDoc]**

[Source code]: https://git.olowe.co/apub
[GoDoc]: https://godoc.org/olowe.co/apub

---

The most popular systems on the [ActivityPub]-speaking [Fediverse] are imitations of non-federated platforms.
[Mastodon] and [Misskey] are [Twitter] clones;
[Lemmy] is a [Reddit] clone.
But ActivityPub –
the protocol connecting all these systems together –
 is often said to be similar to email,
which involves exchanging messages.
In the case of at least Mastodon and Lemmy,
ActivityPub was implemented after the bulk of each sofware was designed.
Message exchange – federation by ActivityPub – is arguably a second-class citizen for these
traditional [CRUD] web applications backed by SQL databases and fronted by web browser UIs.

Apas is an experiment in exposing ActivityPub in a familiar and popular interface: email.
Its primary goal is to clarify how ActivityPub and the Fediverse work for the broader community.
A number of secondary goals are detailed later.

[CRUD]: https://en.wikipedia.org/wiki/Create,_read,_update_and_delete
[upas]: http://doc.cat-v.org/bell_labs/upas_mail_system/
## 1. Motivation

As a fan of [Plan 9] and a weirdo who likes to fiddle with network protocols for fun,
I was disappointed with what using Mastodon, Lemmy et al. felt like.

What excites me is *communication!*
Exchanging messages *between* people, systems, and places we can't think of yet!
It's what makes receiving even just a single email from a random person such a viscerally distinct experience from
thousands reading your post you uploaded somewhere.
We're communicating!

Implementing a subset of the Mastodon and Lemmy HTTP APIs in a couple of languages was relatively straightforward.
After writing some small clients and tooling
it felt like I was just dealing with platforms,
not a federated universe.
The pattern was familiar for many software developers:

* You create a post,
* that gets written to a database,
* you get an ID back, indicating success.

But the whole federation bit is obscured.
You hope that others can see that post... somehow...?
The platform thinking is evident in the language we see around these systems:
"I saw this on Lemmy", or "this is trending on Mastodon", or "find me on Akkoma".
Nobody says "find me on email" or "someone sent this on email".

Interoperability efforts fall flat when expertise in one system does not translate to another.
Moderation and tooling discussions are artificially limited to a particular system.
Should a plugin for Friendica filtering posts containing racist language only work with Friendica when all the systems work together?
Should it even be a plugin tied to one particular system in a particular programming language at all?

Finally, interoperability and portability is the "killer feature" of ActivityPub systems and any significant software system.
We know software developers can write standalone Twitter clones day-in, day-out.
But no amount of funding to Instagram or any other incumbent commercial platform
will ever make it available for
shitty government systems,
space stations,
embedded devices,
your grandmother,
charities,
snail mail,
VR headsets, and
automatic vacuum cleaners
*all at once*.

Writing better software systems often means communicating better.
That means understanding ActivityPub better.

## 2. Overview

**Apas** is mostly inspired by the [upas] email system available with [Plan 9];
a collection of small programs operate on files and streams,
relaying messages out to the Internet or
delivering to mailboxes on the filesystem.
But it's so much more limited and poorly designed than upas that I was hesitant to even write this bit!

### 2.1 Messages

**Apas** marshals ActivityPub objects into [RFC5322] messages and vice-versa.

The [Note Activity] is probably the most recognisable object exchanged by ActivityPub servers.
They are represented as comments by Lemmy, and posts (toots?) by Mastodon.
For instance, imagine a reply from Alex to Bowie talking about motorcycle tyres.
It's passed around as JSON-encoded data like so:

	{
		"type": "Note"
		"id": "https://apub.example.com/alex/12345678",
		"attributedTo": "https://apub.example.com/alex"
		"to": "https://apas.test.example/bowie/87654321",
		"cc": "https://apas.test.example/bowie/followers",
		"inReplyTo": "https://apas.test.example/bowie/87654321",
		"name": "Thoughts on 50/50 tyres"
		"content": "But what if you don't know when you want to ride off-road?",
	}

For **apas** this is equivalent to the mail message:

	From: "Alice " <alice@apub.example.com>
	To: "Bowie" <bowie@apas.test.example>
	CC: "Bowie (followers)" <bowie+followers@apas.test.example>
	Message-ID: <https://apub.example.com/alex/12345678>
	In-Reply-To: <https://apas.test.example/bowie/87654321>
	Subject: Thoughts on 50/50 tyres

	But what if you don't know when you want to ride off-road?

Critically the mail message is written and read by people; not machines.
For developers, administrators, and advanced users, seeing data like
this builds familiarity with the actual data exchanged and behaviour
between systems.

If there was only one thing to take away from **apas**, it's that
familiarity with data over the code is hugely helpful for
troubleshooting and understanding. Especially when typical bug reports
consist of URLs to web apps (or even just screenshots!) trying to
explain what was sent versus what was received. That's before we we
even address what could be in a database, itself requiring its own
query language and tightly-controlled administrative access to read.

[Note Activity]: https://www.w3.org/TR/activitystreams-vocabulary/#dfn-note

### 2.2. Sending

Presenting posts, comments, notes, etc. as a mail message immediately
clarifies a big source of confusion with existing systems:
why isn't my post showing up?
It becomes easier to reason about this when it is obvious where is a message is sent.
An email lists recipients explicitly.
When replying to a Kbin comment via Mastodon,
it takes knowledge of how each system is implemented to know who the
recipients are, if any.

Regular email clients
(or even any old text editor!)
provide an interactive way to test other AP systems.
For instance, we can easily test how the message is received if we address the recipient in the `CC` field instead of `To`,
or if we list the same recipient in both fields. Or 20 times over.
apas could report deliverability errors either:

* immediately, or
* as a bounced message

At the moment, apas returns complete error messages immediately.
This has provided a pleasant enough testing experience
that makes learning ActivityPub an interactive process,
right in the mail client, especially compared with the
usual drudgery of sifting through logs of big web applications.

Sending is comprised two programs each playing its own role:
a submission program accepts messages from authorised users,
and a mailer handles sending messages to the Internet.

![](send.png)

#### 2.2.1. Submission

Messages are submitted to a server running `apsubmit`.
`apsubmit` is a SMTP server.
It listens for SMTP connections,
authenticates the session,
then passes the received message to the mailer `apsend`.

SMTP is a widely implemented protocol.
`apsubmit` enables
existing mail clients,
embededed devices,
and systems that I don't even know exist,
to publish to the Fediverse.

For personal use,
it has been fine using [mutt] via SSH on a Linux server,
[Sylpheed] on my OpenBSD laptop,
[MailMate] on a shared iMac,
and the built-in Mail app on my iPhone for replies.
I'll leave others to come up with more ideas;
keep in mind weather stations, printers, video records can usually
send email but definitely cannot speak ActivityPub!

In the interest of fast feedback,
`apsubmit` takes advantage of the `RCPT` stage of the SMTP transaction.
It verifies that listed recipients exist and have inboxes we can target.
This is in contrast with e.g. Mastodon,
which will always accept creating the following post:

	@john@nowhere.invalid @deleteduser@example.org what do you think?

There's several possible error conditions here. For `john`, perhaps:

* their server is down and messages are undeliverable,
* their Actor is misconfigured and is missing an `inbox` endpoint,
* the address is totally invalid

For `deleteduser`, perhaps the account no longer exists.
Mastodon never notifies of any delivery errors.
We could ask the server administrators to trawl through the server's logs for us,
or ask `johnny` and `deleteduser` out-of-band if they got our message.
Accounting for these types of error at submission time obviates all that extra work.

#### 2.2.2 Mailer

Sending messages is performed by a command-line utility called `apsend`.
`apsend` reads a message from standard input and disposes of it based on the recipients.
If the above message was in a file called "note.eml",
we could send it with the following shell command:

	apsend -t < file.eml

`apsend` is not intended to be executed directly by users.

### 2.3 Receiving

`apserve` provides a typical HTTP server for a minimal ActivityPub service.
It is responsible for:

* receiving Activity over HTTP (ActivityPub inbox)
* serving users' sent Activity for other servers to fetch (ActivityPub outbox)
* serving each user's Actor
* resolving WebFinger lookups

![](receive.png)

Delivery is not handled by `apserve`.
Instead, `apserve` converts Activities to mail messages,
and passes them on to `apsend` for delivery.

#### 2.3.1 Filtering, spam


## 3. Workarounds & limitations

The mapping between Activity objects, mail messages,
ActivityPub HTTP methods, and SMTP transactions
has a number of limitations.
**Apas** uses some workarounds internally to fill some gaps.

The [Mention] Activity,
used by Mastodon for notifications,
is implemented by reading the To field of submitted messages.
Recipients in `To` are added as Mentions.
For example, the message:

	To: "Oliver Lowe" <otl@hachyderm.io>

	Hello!

results in an entry in `tags` in an ActivityPub Note:

	{
		"type": "Note"
		...
		"tags": {[
			"type": "Mention",
			"href": "https://hachyderm.io/users/otl",
			"name": "@otl@hachyderm.io"
		]}
	}

There is not an easy way to address an Actor's followers using the `acct:` mail address syntax.
`apas` understands a non-standard syntax using "plus addressing".
For example to address the followers of user@example.com
the address user+followers@example.com may be used.
These followers addresses cannot be resolved by WebFinger.

Likes and Dislikes are silently dropped by `apserve`.
The reader can decide whether this is a workaround, feature, or bug.

To simplifly delivery to local mailboxes,
Actors served by `apserve` have no shared inbox/outbox.
Fortunately shared inbox endpoints are inteded as a performance opimitisation for
servers hosting many Actors, which is beyond the scope of **apas**.

[Zawinski's Law of Software Envelopment]: https://en.wikipedia.org/wiki/Jamie_Zawinski#Zawinski's_Law
[NetNewsWire]: https://netnewswire.com
[mutt]: http://www.mutt.org
[MailMate]: https://freron.com
[Sylpheed]: https://sylpheed.sraoss.jp/en/

[ActivityPub]: https://en.wikipedia.org/wiki/ActivityPub
[Misskey]: https://misskey-hub.net
[Twitter]: https://en.wikipedia.org/wiki/Twitter
[Reddit]: https://en.wikipedia.org/wiki/Reddit

[Mention]: https://www.w3.org/TR/activitystreams-vocabulary/#microsyntaxes

[2023 Reddit API controversy]: https://en.wikipedia.org/wiki/2023_Reddit_API_controversy
[3rd Party Twitter Apps]: https://www.theverge.com/2023/1/22/23564460/twitter-third-party-apps-history-contributions
[Lemmy]: https://join-lemmy.org
[Mastodon]: https://joinmastodon.org
[RFC5322]: https://www.rfc-editor.org/rfc/rfc5322
[Plan 9]: http://9p.io/plan9/
