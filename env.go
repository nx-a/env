package env

import (
	"embed"
	"fmt"
	"github.com/nx-a/conv"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

func Get[T any](env *Environment, name string) T {
	content := GetInterface(env, name)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()
	return conv.To[T](env.getenv(content))
}

func (env *Environment) getenv(val interface{}) interface{} {
	if _, ok := val.(string); !ok {
		return val
	}
	value := val.(string)
	if isEnv.MatchString(value) {
		find := isEnv.FindStringSubmatch(value)
		sub := strings.SplitN(find[1], ":", 2)
		envOs := os.Getenv(sub[0])
		if len(envOs) > 0 {
			return envOs
		}
		if len(sub) > 1 {
			return sub[1]
		}
		return nil
	}
	return val
}

type Environment struct {
	mutex sync.RWMutex
	data  map[string]interface{}
}

var isEnv = regexp.MustCompile(`\$\{(.*?)\}`)

func New(emb embed.FS) *Environment {
	env := &Environment{
		data: make(map[string]interface{}),
	}
	env.mutex.Lock()
	defer env.mutex.Unlock()
	env.data = convertYamlToProp([]byte(def))
	file, err := emb.ReadFile("config.yml")
	if err != nil {
		return nil
	}
	subenv := convertYamlToProp(file)
	for key, value := range subenv {
		env.data[key] = value
	}
	return env
}
func (env *Environment) Add(path string) {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	subenv := convertYamlToProp(file)
	for key, value := range subenv {
		env.data[key] = value
	}
}

var def = `server:
  host: ${SERVER_HOST:*}
  port: ${SERVER_PORT:':80'}
  maxSize: ${MAX_REQUEST_BODY_SIZE:104857600}
service:
  name: ${SERVICE_NAME:app}
  prod: ${PROD:false}`

func GetInterface(env *Environment, name string) interface{} {
	subName := strings.Split(name, ".")
	data := env.data
	var val interface{}
	var ok bool
	for i, nameData := range subName {
		val, ok = data[nameData]
		if !ok {
			return nil
		}
		if i < len(subName)-1 {
			typeReflect := reflect.TypeOf(val)
			if typeReflect.Kind() == reflect.Map {
				data = val.(map[string]interface{})
			} else {
				return nil
			}
		}
	}
	return val
}
func (e *Environment) Get(name string) string {
	return GetString(e, name)
}
func GetString(env *Environment, name string) string {
	val := GetInterface(env, name)
	if val == nil {
		return ""
	}
	if reflect.TypeOf(val).Kind() != reflect.String {
		return conv.To[string](val)
	}
	strVal := val.(string)
	return checkEnv(strVal)
}
func checkEnv(value string) string {
	if isEnv.MatchString(value) {
		find := isEnv.FindStringSubmatch(value)
		sub := strings.SplitN(find[1], ":", 2)
		envOs := os.Getenv(sub[0])
		if len(envOs) > 0 {
			return envOs
		}
		if len(sub) > 1 {
			return sub[1]
		}
	}
	return strings.TrimSpace(value)
}

func convertYamlToProp(file []byte) map[string]interface{} {
	var local map[string]interface{}
	err := yaml.Unmarshal(file, &local)
	if err != nil {
		return nil
	}
	return local
}
