/*
Command apsend reads a mail message from the standard input
and disposes of it based on the provided recipient addresses.

Its usage is:

	apsend [ -F ] [ -t ] rcpt ...

Messages are disposed of in one of two ways:

  - If the recipient refers to a local user, the message is appended to their local mailbox.
  - If the recipients refers to a remote user or collection, the message is sent via ActivityPub to each recipent's corresponding Actor inbox.

Local recipients are addrsessed as a plain username such as "otl".
Remote recipients are addressed as an email address,
such as
"mort@novum.streats.dev" or
"otl+followers@hachyderm.io".

apsend is not intended to be executed directly by users.
Usually it is executed as a mailer by a SMTP server like [apsubmit],
or by a server which receives ActivityPub activities for local recipients like [apserve].

The flags understood are:

  - *-F* File a copy to the sender's mailbox.

  - *-t* Read recipients from the To: and CC: lines of the message.

# Example

Given the following message in the file greeting.eml:

	From: otl@apubtest2.srcbeat.com
	To: otl@hachyderm.io

	Hello, Oliver!

Send it with the following command:

	apsend otl@hachyderm.io < greeting.eml
*/
package main
