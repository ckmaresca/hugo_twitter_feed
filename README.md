# hugo_twitter_feed
Go hack to get a Twitter feed into Hugo as individual posts

This will pull Twitter posts and turn them into individual files in the directory 'content'.  

Each file will have the Twitter ID of the post in the filename and a file named 'last_id' is used to keep track of the last Twitter post pulled so that incremental downloads can be done.

The point of all this is to generate Markdown formatted post for the Hugo CMS to use.

Note that this is just my crappy code for my own personal website - use at your own risk.
