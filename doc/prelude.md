---

### 1.1. Motivation, Communication

I signed up for Twitter account back in 2007.
I was 15 years old and didn't really get it.
All I knew was that the people whose blogs I subscribed to via RSS were now also *microblogging*.
Since that time I've doubled in age and still don't get it.
But I *still* use
[NetNewsWire],
[mutt],
[Sylpheed],
[MailMate],
and Apple's Mail.app
across OpenBSD, Linux, macOS and iOS.

Reddit and Twitter lost the plot around the same time
(see [2023 Reddit API controversy] and [3rd Party Twitter Apps]).
No big deal.

Here and there I would read posts to smaller communities on Reddit (via RSS)
and read the comment threads (`old.reddit.com` forever).
Links to Twitter get thrown around sometimes too.
But I knew us nerds wouldn't settle until we had our artisanally crafted apps back.
What would we come up with next?

I'd heard of [Mastodon] before,
maybe [Lemmy], too.
I looked at screenshots of the default interfaces and immediately thought "nope".

As a lover of [Plan 9]
and weirdo who mucks around with network protocols for fun,
platforms are boring.
What excites me is *communication*!
Exchanging messages *between* people, systems, and places we can't think of yet!
It's what makes receiving even just a single email from a random person such a viscerally distinct experience from
thousands reading your post you uploaded somewhere.
We're communicating!

Then I heard that Lemmy can interact with Mastodon.
"That's cool", I thought.
But nothing excited me more when I read this about how it actually worked:

> ActivityPub is a lot like email.
> You post stuff to inboxes and people read your outbox.

Maybe these new systems weren't platforms after all.
Maybe there was no platform.

### 1.2 Lemmy API



### 1.2 Mastodon API

It started with a script.
`mastodump` read my timeline via the Mastodon HTTP API,
then wrote each status as a [RFC5322] message – an email – to the filesystem:

	for status in timeline():
		if os.path.isfile(status.id):
			continue
		with open(status.id) as f:
			print("From: <%s>\n" % status.owner, file=f)
			print("Subject: %s\n" % status.body, file=f)
			print("Message-ID: <%d>\n" % status.id, file=f)
			print("\n", file=f)
			print(status.body, file=f)

`cron` executed `mastodump` every 5 minutes for me.

To create posts, I wrote `mastopost`.
It was a small SMTP server which received messages over SMTP and created posts to Mastodon:

	laptop --SMTP--> mastopost --HTTP--> Mastodon --> fediverse

---

The problem with both of these approaches is that what I'm really doing is
I'm not really communicating with the world.
Instead, I'm uploading some data to a platform.
My client keeps track of the state of my little platform (my Mastodon/Lemmy instance).
My instance manages communication with the outside world for me.

