package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudflare/cloudflare-go"
	"github.com/digitalocean/godo"
	"github.com/spf13/viper"
	"github.com/vatsimnetwork/vatdns/internal/logger"
	"gopkg.in/yaml.v3"
	"log"
	"net"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	dnsServers sync.Map
)

type DnsServer struct {
	Name    string `json:"name"`
	Latency int64  `json:"latency"`
	Result  string `json:"result"`
	Pass    bool   `json:"pass"`
}

type retardantFoamDataFile struct {
	IpServerFiles []ipServerFile `yaml:"ipServerFiles"`
}

type ipServerFile struct {
	Name     string `yaml:"name"`
	Region   string `yaml:"region"`
	Bucket   string `yaml:"bucket"`
	Contents string `yaml:"contents"`
}

func failover() {
	ctx := context.Background()
	doSpacesKey := viper.GetString("DO_SPACES_KEY")
	doSpacesSecret := viper.GetString("DO_SPACES_SECRET")
	doSpacesRegion := viper.GetString("DO_SPACES_REGION")
	// Push IP files to Spaces
	retardantFoamData := retardantFoamDataFile{}
	yamlData, err := os.ReadFile("retardantfoam.yaml")
	if err != nil {
		logger.Fatal("Reading retardantfoam.yaml failed")
	}
	_ = yaml.Unmarshal(yamlData, &retardantFoamData)
	for _, v := range retardantFoamData.IpServerFiles {
		creds := credentials.NewStaticCredentialsProvider(doSpacesKey, doSpacesSecret, "")

		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: fmt.Sprintf("https://%s.digitaloceanspaces.com", v.Region),
			}, nil
		})
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(creds),
			config.WithEndpointResolverWithOptions(customResolver))
		if err != nil {
		}
		// Create an Amazon S3 service client
		doSpacesClient := s3.NewFromConfig(cfg)
		_, err = doSpacesClient.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(v.Bucket),
			Key:    aws.String(v.Name),
			Body:   bytes.NewBufferString(v.Contents),
		})
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to upload %s to Spaces", v.Name))
		} else {
			logger.Info(fmt.Sprintf("Uploaded %s to Spaces", v.Name))
		}
	}

	api, err := cloudflare.NewWithAPIToken(viper.GetString("CLOUDFLARE_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	_, err = api.PurgeCache(ctx, viper.GetString("CLOUDFLARE_ZONE_ID"), cloudflare.PurgeCacheRequest{
		Everything: true,
	})
	if err != nil {
		logger.Error("Failed to flush Cloudflare cache.")
	} else {
		logger.Info("Flushed Cloudflare cache.")

		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL: fmt.Sprintf("https://%s.digitaloceanspaces.com", doSpacesRegion),
			}, nil
		})
		creds := credentials.NewStaticCredentialsProvider(doSpacesKey, doSpacesSecret, "")
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(creds),
			config.WithEndpointResolverWithOptions(customResolver))
		if err != nil {
		}
		// Create an Amazon S3 service client
		doSpacesClient := s3.NewFromConfig(cfg)

		_, err = doSpacesClient.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(viper.GetString("DO_SPACES_BUCKET_NAME")),
			Key:    aws.String("vatdns-heartbeat-blocker"),
			Body:   bytes.NewBufferString(""),
		})
		if err != nil {
			logger.Error("Failed to upload heartbeat blocker file to Spaces")
		} else {
			logger.Info("Heartbeat blocker file written to Spaces. Remove it to enable this tool again. Chances you will need to delete vatdns-heartbeat as well.")
		}
	}
}

func main() {
	logger.Info("Reading config")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetDefault("DO_API_KEY", "")
	viper.SetDefault("DO_TAG", "")
	viper.SetDefault("DO_SPACES_KEY", "")
	viper.SetDefault("DO_SPACES_SECRET", "")
	viper.SetDefault("DO_SPACES_REGION", "")
	viper.SetDefault("DO_SPACES_BUCKET_NAME", "")
	viper.SetDefault("CLOUDFLARE_API_KEY", "")
	viper.SetDefault("CLOUDFLARE_LB_ID", "")
	viper.SetDefault("CLOUDFLARE_ZONE_ID", "")
	viper.SetDefault("CLOUDFLARE_ACCOUNT_ID", "")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	var wg = sync.WaitGroup{}
	logger.Info("Reading config")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetDefault("DO_API_KEY", "")
	viper.SetDefault("DO_TAG", nil)
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	logger.Info("retardantfoam - I put out fires...")
	ctx := context.TODO()
	doSpacesKey := viper.GetString("DO_SPACES_KEY")
	doSpacesSecret := viper.GetString("DO_SPACES_SECRET")
	doSpacesRegion := viper.GetString("DO_SPACES_REGION")
	failoverTimeLimit := viper.GetInt64("FAILOVER_TIME_LIMIT")
	creds := credentials.NewStaticCredentialsProvider(doSpacesKey, doSpacesSecret, "")

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.digitaloceanspaces.com", doSpacesRegion),
		}, nil
	})
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
		config.WithEndpointResolverWithOptions(customResolver))
	if err != nil {
	}
	// Create an Amazon S3 service client
	doSpacesClient := s3.NewFromConfig(cfg)

	_, err = doSpacesClient.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(viper.GetString("DO_SPACES_BUCKET_NAME")),
		Key:    aws.String("vatdns-heartbeat-blocker"),
	})
	if err != nil {
		var re s3.ResponseError
		if errors.As(err, &re) {
			if strings.Contains(re.Error(), "StatusCode: 404") {
			} else {
				logger.Fatal("Unable contact Spaces.")
			}
		}
	} else {
		logger.Fatal("Heartbeat blocker file exists in Spaces. Operation disabled. Please remove to enable.")
	}

	// Check for heartbeat file
	heartbeatObj, err := doSpacesClient.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(viper.GetString("DO_SPACES_BUCKET_NAME")),
		Key:    aws.String("vatdns-heartbeat"),
	})
	if err != nil {
		var re s3.ResponseError
		if errors.As(err, &re) {
			if strings.Contains(re.Error(), "StatusCode: 404") {
				logger.Info("Initial file check 404, first run? This is usually fine.")
			} else {
				logger.Fatal("Unable contact Spaces.")
			}
		}
	} else {
		timeSinceLastHeartbeat := time.Now().Unix() - heartbeatObj.LastModified.Unix()
		if timeSinceLastHeartbeat >= failoverTimeLimit {
			logger.Error("Over failover time limit reached, pushing IP server list and flushing Cloudflare cache.")
			failover()
			os.Exit(0)
		} else {
			logger.Info(fmt.Sprintf("Failover time limit not reached. Currently %d seconds since last heartbeat.", timeSinceLastHeartbeat))
		}
	}

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	doClient := godo.NewFromToken(viper.GetString("DO_API_KEY"))
	droplets, _, _ := doClient.Droplets.ListByTag(ctx, viper.GetString("DO_TAG"), opt)
	logger.Info("Checked tag for Droplets")
	for _, d := range droplets {
		logger.Info(fmt.Sprintf("Checking DNS health of %s", d.Name))
		d := d
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now().UnixNano() / int64(time.Millisecond)
			r := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					dial := net.Dialer{
						Timeout: time.Second * 1,
					}
					publicIPv4, _ := d.PublicIPv4()
					return dial.DialContext(ctx, network, fmt.Sprintf("%s:%s", publicIPv4, viper.GetString("DNS_PORT")))
				},
			}
			ip, err := r.LookupHost(context.Background(), "fsd.connect.vatsim.net")
			finish := time.Now().UnixNano() / int64(time.Millisecond)
			diff := finish - start
			if err == nil {
				logger.Info(fmt.Sprintf("Successfully queried %s: %dms", d.Name, diff))
				dnsServers.Store(d.Name, DnsServer{
					Name:    d.Name,
					Latency: diff,
					Result:  ip[0],
					Pass:    true,
				})
			} else {
				logger.Error(fmt.Sprintf("Error when checking %s: %s", d.Name, err))
				dnsServers.Store(d.Name, DnsServer{
					Name:    d.Name,
					Latency: 0.0,
					Result:  "",
					Pass:    false,
				})
			}
		}()
	}
	wg.Wait()
	atLeastOneHealthy := false

	dnsHealthyCounter := 0
	dnsHealthyCounter = 0
	dnsServers.Range(func(k, v interface{}) bool {
		fsdServerStruct := v.(DnsServer)
		if fsdServerStruct.Pass == true {
			atLeastOneHealthy = true
			dnsHealthyCounter += 1
		}
		return true
	})
	if atLeastOneHealthy {
		logger.Info(fmt.Sprintf("DNS is working. %d DNS servers healthy. Writing heartbeat file to Spaces.", dnsHealthyCounter))

		_, err = doSpacesClient.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(viper.GetString("DO_SPACES_BUCKET_NAME")),
			Key:    aws.String("vatdns-heartbeat"),
			Body:   bytes.NewBufferString(""),
		})
		if err == nil {
			logger.Info("Wrote heartbeat file to Spaces")
		} else {
			logger.Info("Failed writing heartbeat file to Spaces")
			// Maybe post to Slack or something???
		}
	} else {
		logger.Info("No healthy DNS servers.")
	}
	os.Exit(0)
}
