Pod::Spec.new do |spec|
  spec.name         = 'Riftcmd'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/cryptorift/riftcore'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS CryptoRift Client'
  spec.source       = { :git => 'https://github.com/cryptorift/riftcore.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Riftcmd.framework'

	spec.prepare_command = <<-CMD
    curl https://riftcmdstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Riftcmd.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
