Pod::Spec.new do |spec|
  spec.name         = 'Highcoin'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/420integrated/go-highcoin'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Highcoin Client'
  spec.source       = { :git => 'https://github.com/420integrated/go-highcoin.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Highcoin.framework'

	spec.prepare_command = <<-CMD
    curl https://highcoinstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Highcoin.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
