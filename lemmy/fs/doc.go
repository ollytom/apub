/*
FS is a read-only filesystem interface to a Lemmy instance.
The root of the filesystem holds directories for each community known to the filesystem.
Local communities are named by their plain name verbatim.
Remote communities have the instance address as a suffix. For example:

	golang/
	plan9@lemmy.sdf.org/
	openbsd@lemmy.sdf.org/

Each community directory holds posts.
Each post has associated a directory numbered by its ID.
Within each post are the following entries:

	body     Text describing, or accompanying, the post.
	creator  The numeric user ID of the post's author.
	title    The post's title.
	url      A URL pointing to a picture or website, usually as the
	         subject of the post if present.
	123...   Numbered files containing user discussion.
	         Described in more detail below.

A comment file is named by its unique comment ID.
Its contents are a RFC 5322 message.
The message body contains the text content of the comment.
The header contains the following fields:

	From       User ID of the comment's author.
	References A list of comment IDs referenced by this comment, one
	           per line. The first line is the immediately referenced
	           comment (the parent); the second is the grandparent and
	           so on. This can be used by readers to render discussion
	           threads.

FS satisfies io/fs.FS.
*/
package fs
