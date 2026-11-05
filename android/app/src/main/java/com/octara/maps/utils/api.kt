package com.octara.maps.utils

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.google.gson.TypeAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter
import com.octara.maps.types.ParentStop
import com.octara.maps.types.Stop
import com.octara.maps.types.Route
import com.octara.maps.types.Departure
import com.octara.maps.types.StopType
import com.octara.maps.types.RouteType
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

class Api {
    companion object {
        const val BASE_URL = "https://omaps-calc.orionhost.app/api"
    }

    val gson: Gson = GsonBuilder()
        .registerTypeAdapter(StopType::class.java, object : TypeAdapter<StopType>() {
            override fun write(out: JsonWriter, value: StopType?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): StopType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return StopType.UNKNOWN
                }
                val intVal = reader.nextInt()
                return StopType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(RouteType::class.java, object : TypeAdapter<RouteType>() {
            override fun write(out: JsonWriter, value: RouteType?) {
                out.value(value?.value ?: 3)
            }

            override fun read(reader: JsonReader): RouteType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return RouteType.BUS
                }
                val intVal = reader.nextInt()
                return RouteType.fromValue(intVal)
            }
        })
        .create()

    inline fun <reified T> fetchGET(url: String): T {
        val connection = (URL(url).openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10_000
            readTimeout = 10_000
        }

        return try {
            val responseCode = connection.responseCode
            if (responseCode !in 200..299) {
                throw IllegalStateException("GET $url failed ($responseCode)")
            }

            val json = connection.inputStream.bufferedReader().use { it.readText() }
            val type = object : com.google.gson.reflect.TypeToken<T>() {}.type
            gson.fromJson<T>(json, type)
                ?: throw IllegalStateException("json is null")
        } finally {
            connection.disconnect()
        }
    }

    private data class StopResponse(
        val total: Int,
        val stops: List<ParentStop>?
    )

    suspend fun searchStopsByName(
        name: String,
        limit: Int = 10,
        cpos: Pair<Float, Float>
    ): List<ParentStop> = withContext(Dispatchers.IO) {
        val encodedQuery = URLEncoder.encode(name, "UTF-8")
        val url = "$BASE_URL/stops/autocomplete?q=$encodedQuery&maxstops=$limit&lat=${cpos.first}&lon=${cpos.second}"

        val response: StopResponse = fetchGET(url)
        response.stops ?: emptyList()
    }

    public data class StopInfoResponse(
        val stop: Stop,
        val departures: List<Departure>?,
        val routes: List<Route>?
    )

    suspend fun getStopById(
        id: String,
    ): StopInfoResponse = withContext(Dispatchers.IO) {
        val url = "$BASE_URL/stops/$id"

        val response: StopInfoResponse = fetchGET(url)
        response
    }

    suspend fun getNextDeparturesOfMultipleStops(list: List<Pair<String, String>>): Map<Pair<String, String>, Map<String, List<Int>>> =
        withContext(Dispatchers.IO) {
            if (list.isEmpty()) return@withContext emptyMap()
            try {
                val url = "$BASE_URL/nextdepartures?stops=${list.joinToString(",") { "${it.first}:${it.second}" }}"
                val response: Map<String, Map<String, List<Any>>> = fetchGET(url)
                val result = mutableMapOf<Pair<String, String>, Map<String, List<Int>>>()

                for ((key, deps) in response) {
                    val clean = key.removePrefix("stop_point:IDFM:").removePrefix("IDFM:")
                    val parts = clean.split(":")
                    if (parts.size >= 2) {
                        val stopId = parts[0]
                        val routeId = parts[1]
                        val sanitizedDeps = deps.mapValues { (_, times) ->
                            times.mapNotNull { item ->
                                when (item) {
                                    is Number -> item.toInt()
                                    is String -> item.toIntOrNull()
                                    else -> null
                                }
                            }
                        }
                        result[stopId to routeId] = sanitizedDeps
                        result["IDFM:$stopId" to "IDFM:$routeId"] = sanitizedDeps
                    }
                }

                for (pair in list) {
                    val cleanPair = pair.first.removePrefix("IDFM:") to pair.second.removePrefix("IDFM:")
                    if (result.containsKey(cleanPair)) {
                        result[pair] = result[cleanPair]!!
                    }
                }
                result
            } catch (e: Exception) {
                e.printStackTrace()
                emptyMap()
            }
        }
}