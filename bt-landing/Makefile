deploy:
	s3cmd sync --delete-removed --no-mime-magic --recursive ./ --acl-public s3://boardthreads.com/ --exclude=.git/* --exclude=*.swp
