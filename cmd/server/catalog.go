package main

import (
	"painel-economico/internal/bcb"
	"painel-economico/internal/ibge"
	"painel-economico/internal/indicator"
)

// catalog lists the indicators the API serves, in display order.
func catalog(sgs *bcb.Client, sidra *ibge.Client) []indicator.Definition {
	return []indicator.Definition{
		{
			Indicator: indicator.Indicator{
				ID:          "selic",
				Name:        "Taxa Selic (meta)",
				ShortName:   "Selic",
				Unit:        "% a.a.",
				Frequency:   indicator.Daily,
				Description: "Meta da taxa básica de juros definida pelo Copom, em % ao ano.",
				Source:      "Banco Central do Brasil, SGS 432",
				SourceURL:   "https://dadosabertos.bcb.gov.br/dataset/432-taxa-de-juros---meta-selic-definida-pelo-copom",
			},
			Upstream: sgs.Series(432),
		},
		{
			Indicator: indicator.Indicator{
				ID:        "ipca-12m",
				Name:      "IPCA acumulado em 12 meses",
				ShortName: "IPCA 12 meses",
				// A 12-month accumulated rate is an annual rate, so it shares
				// the Selic unit and the two can be read on the same axis.
				Unit:        "% a.a.",
				Frequency:   indicator.Monthly,
				Description: "Inflação oficial acumulada nos últimos 12 meses.",
				Source:      "IBGE, via Banco Central (SGS 13522)",
				SourceURL:   "https://sidra.ibge.gov.br/tabela/1737",
			},
			Upstream: sgs.Series(13522),
		},
		{
			Indicator: indicator.Indicator{
				ID:          "ipca",
				Name:        "IPCA (variação mensal)",
				ShortName:   "IPCA mensal",
				Unit:        "% no mês",
				Frequency:   indicator.Monthly,
				Description: "Variação dos preços ao consumidor em cada mês.",
				Source:      "IBGE, via Banco Central (SGS 433)",
				SourceURL:   "https://sidra.ibge.gov.br/tabela/1737",
			},
			Upstream: sgs.Series(433),
		},
		{
			Indicator: indicator.Indicator{
				ID:          "dolar",
				Name:        "Dólar (PTAX venda)",
				ShortName:   "Dólar",
				Unit:        "R$",
				Frequency:   indicator.Daily,
				Description: "Taxa de câmbio PTAX de venda, em reais por dólar, nos dias úteis.",
				Source:      "Banco Central do Brasil, SGS 1",
				SourceURL:   "https://dadosabertos.bcb.gov.br/dataset/1-taxa-de-cambio---livre---dolar-americano-venda---diario",
			},
			Upstream: sgs.Series(1),
		},
		{
			Indicator: indicator.Indicator{
				ID:          "desemprego",
				Name:        "Taxa de desemprego",
				ShortName:   "Desemprego",
				Unit:        "%",
				Frequency:   indicator.Monthly,
				Description: "Taxa de desocupação da PNAD Contínua no trimestre móvel que termina no mês.",
				Source:      "IBGE, PNAD Contínua (SIDRA 6381)",
				SourceURL:   "https://sidra.ibge.gov.br/tabela/6381",
			},
			Upstream: sidra.Variable(6381, 4099),
		},
	}
}
