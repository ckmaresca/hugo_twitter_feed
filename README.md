# Hugo Twitter Feed
Go hack to get a Twitter feed into Hugo as individual posts

This will pull Twitter posts and turn them into individual files in the directory 'content'.  

Each file will have the Twitter ID of the post in the filename and a file named 'last_id' is used to keep track of the last Twitter post pulled so that incremental downloads can be done.

The point of all this is to generate Markdown formatted post for the Hugo CMS to use.  Conversion from JSON to Markdown is still a work in progress.

Uses the following packages:

"github.com/kurrik/oauth1a"

"github.com/kurrik/twittergo"

This is basically a heavily modified version of the 'user timeline' example from TwitterGo

Note that this is just my crappy code for my own personal website - use at your own risk.
