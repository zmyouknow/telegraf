package config

import (
	"testing"
	"time"

	"github.com/influxdb/telegraf/plugins"
	"github.com/influxdb/telegraf/plugins/exec"
	"github.com/influxdb/telegraf/plugins/memcached"
	"github.com/influxdb/telegraf/plugins/procstat"
	"github.com/stretchr/testify/assert"
)

func TestConfig_LoadSinglePlugin(t *testing.T) {
	c := NewConfig()
	c.LoadConfig("./testdata/single_plugin.toml")

	memcached := plugins.Plugins["memcached"]().(*memcached.Memcached)
	memcached.Servers = []string{"localhost"}

	mConfig := &PluginConfig{
		Name: "memcached",
		Filter: Filter{
			Drop: []string{"other", "stuff"},
			Pass: []string{"some", "strings"},
			TagDrop: []TagFilter{
				TagFilter{
					Name:   "badtag",
					Filter: []string{"othertag"},
				},
			},
			TagPass: []TagFilter{
				TagFilter{
					Name:   "goodtag",
					Filter: []string{"mytag"},
				},
			},
			IsActive: true,
		},
		Interval: 5 * time.Second,
	}

	assert.Equal(t, memcached, c.Plugins[0].Plugin,
		"Testdata did not produce a correct memcached struct.")
	assert.Equal(t, mConfig, c.Plugins[0].Config,
		"Testdata did not produce correct memcached metadata.")
}

func TestConfig_LoadDirectory(t *testing.T) {
	c := NewConfig()
	err := c.LoadConfig("./testdata/single_plugin.toml")
	if err != nil {
		t.Error(err)
	}
	err = c.LoadDirectory("./testdata/subconfig")
	if err != nil {
		t.Error(err)
	}

	memcached := plugins.Plugins["memcached"]().(*memcached.Memcached)
	memcached.Servers = []string{"localhost"}

	mConfig := &PluginConfig{
		Name: "memcached",
		Filter: Filter{
			Drop: []string{"other", "stuff"},
			Pass: []string{"some", "strings"},
			TagDrop: []TagFilter{
				TagFilter{
					Name:   "badtag",
					Filter: []string{"othertag"},
				},
			},
			TagPass: []TagFilter{
				TagFilter{
					Name:   "goodtag",
					Filter: []string{"mytag"},
				},
			},
			IsActive: true,
		},
		Interval: 5 * time.Second,
	}
	assert.Equal(t, memcached, c.Plugins[0].Plugin,
		"Testdata did not produce a correct memcached struct.")
	assert.Equal(t, mConfig, c.Plugins[0].Config,
		"Testdata did not produce correct memcached metadata.")

	ex := plugins.Plugins["exec"]().(*exec.Exec)
	ex.Commands = []*exec.Command{
		&exec.Command{
			Command: "/usr/bin/myothercollector --foo=bar",
			Name:    "myothercollector",
		},
	}
	eConfig := &PluginConfig{Name: "exec"}
	assert.Equal(t, ex, c.Plugins[1].Plugin,
		"Merged Testdata did not produce a correct exec struct.")
	assert.Equal(t, eConfig, c.Plugins[1].Config,
		"Merged Testdata did not produce correct exec metadata.")

	memcached.Servers = []string{"192.168.1.1"}
	assert.Equal(t, memcached, c.Plugins[2].Plugin,
		"Testdata did not produce a correct memcached struct.")
	assert.Equal(t, mConfig, c.Plugins[2].Config,
		"Testdata did not produce correct memcached metadata.")

	pstat := plugins.Plugins["procstat"]().(*procstat.Procstat)
	pstat.Specifications = []*procstat.Specification{
		&procstat.Specification{
			PidFile: "/var/run/grafana-server.pid",
		},
		&procstat.Specification{
			PidFile: "/var/run/influxdb/influxd.pid",
		},
	}

	pConfig := &PluginConfig{Name: "procstat"}

	assert.Equal(t, pstat, c.Plugins[3].Plugin,
		"Merged Testdata did not produce a correct procstat struct.")
	assert.Equal(t, pConfig, c.Plugins[3].Config,
		"Merged Testdata did not produce correct procstat metadata.")
}

func TestFilter_Empty(t *testing.T) {
	f := Filter{}

	measurements := []string{
		"foo",
		"bar",
		"barfoo",
		"foo_bar",
		"foo.bar",
		"foo-bar",
		"supercalifradjulisticexpialidocious",
	}

	for _, measurement := range measurements {
		if !f.ShouldPass(measurement) {
			t.Errorf("Expected measurement %s to pass", measurement)
		}
	}
}

func TestFilter_Pass(t *testing.T) {
	f := Filter{
		Pass: []string{"foo*", "cpu_usage_idle"},
	}

	passes := []string{
		"foo",
		"foo_bar",
		"foo.bar",
		"foo-bar",
		"cpu_usage_idle",
	}

	drops := []string{
		"bar",
		"barfoo",
		"bar_foo",
		"cpu_usage_busy",
	}

	for _, measurement := range passes {
		if !f.ShouldPass(measurement) {
			t.Errorf("Expected measurement %s to pass", measurement)
		}
	}

	for _, measurement := range drops {
		if f.ShouldPass(measurement) {
			t.Errorf("Expected measurement %s to drop", measurement)
		}
	}
}

func TestFilter_Drop(t *testing.T) {
	f := Filter{
		Drop: []string{"foo*", "cpu_usage_idle"},
	}

	drops := []string{
		"foo",
		"foo_bar",
		"foo.bar",
		"foo-bar",
		"cpu_usage_idle",
	}

	passes := []string{
		"bar",
		"barfoo",
		"bar_foo",
		"cpu_usage_busy",
	}

	for _, measurement := range passes {
		if !f.ShouldPass(measurement) {
			t.Errorf("Expected measurement %s to pass", measurement)
		}
	}

	for _, measurement := range drops {
		if f.ShouldPass(measurement) {
			t.Errorf("Expected measurement %s to drop", measurement)
		}
	}
}

func TestFilter_TagPass(t *testing.T) {
	filters := []TagFilter{
		TagFilter{
			Name:   "cpu",
			Filter: []string{"cpu-*"},
		},
		TagFilter{
			Name:   "mem",
			Filter: []string{"mem_free"},
		}}
	f := Filter{
		TagPass: filters,
	}

	passes := []map[string]string{
		{"cpu": "cpu-total"},
		{"cpu": "cpu-0"},
		{"cpu": "cpu-1"},
		{"cpu": "cpu-2"},
		{"mem": "mem_free"},
	}

	drops := []map[string]string{
		{"cpu": "cputotal"},
		{"cpu": "cpu0"},
		{"cpu": "cpu1"},
		{"cpu": "cpu2"},
		{"mem": "mem_used"},
	}

	for _, tags := range passes {
		if !f.ShouldTagsPass(tags) {
			t.Errorf("Expected tags %v to pass", tags)
		}
	}

	for _, tags := range drops {
		if f.ShouldTagsPass(tags) {
			t.Errorf("Expected tags %v to drop", tags)
		}
	}
}

func TestFilter_TagDrop(t *testing.T) {
	filters := []TagFilter{
		TagFilter{
			Name:   "cpu",
			Filter: []string{"cpu-*"},
		},
		TagFilter{
			Name:   "mem",
			Filter: []string{"mem_free"},
		}}
	f := Filter{
		TagDrop: filters,
	}

	drops := []map[string]string{
		{"cpu": "cpu-total"},
		{"cpu": "cpu-0"},
		{"cpu": "cpu-1"},
		{"cpu": "cpu-2"},
		{"mem": "mem_free"},
	}

	passes := []map[string]string{
		{"cpu": "cputotal"},
		{"cpu": "cpu0"},
		{"cpu": "cpu1"},
		{"cpu": "cpu2"},
		{"mem": "mem_used"},
	}

	for _, tags := range passes {
		if !f.ShouldTagsPass(tags) {
			t.Errorf("Expected tags %v to pass", tags)
		}
	}

	for _, tags := range drops {
		if f.ShouldTagsPass(tags) {
			t.Errorf("Expected tags %v to drop", tags)
		}
	}
}
