Vagrant.configure(2) do |config|
	config.vm.box = "ubuntu/trusty32"
	config.vm.provision "shell", inline: <<-SHELL
		apt-get update || exit
		apt-get install -y build-essential git\
			redis-server\
			libpng12-dev\
			imagemagick\
			software-properties-common\
			pngquant\
		|| exit

		add-apt-repository ppa:mc3man/trusty-media -y || exit
		apt-get update || exit
		apt-get install ffmpeg -y || exit

		echo "Installing io.js..."
		wget -O - -q https://iojs.org/dist/v2.0.2/iojs-v2.0.2-linux-x86.tar.xz | \
			tar xpJ -C /usr/local --strip=1 || exit

		# Deal with debian bullshit
		ln -sf /usr/local/bin/node /usr/bin/

		# cd to meguca's root on login
		echo 'cd /vagrant' >> /etc/profile
		
		echo "Installing npm modules..."
		su vagrant -
			cd /vagrant
			npm install --unsafe-perm
		exit
	SHELL
	# Server
	config.vm.network :forwarded_port, host: 8000, guest: 8000
	# JSON API
	config.vm.network :forwarded_port, host: 8002, guest: 8002
	# Node debug port
	config.vm.network :forwarded_port, host: 5858, guest: 5858
end
