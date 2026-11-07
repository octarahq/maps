package com.octara.maps.ui.screens.map

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.Place
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.StarOutline
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SheetState
import androidx.compose.material3.SheetValue
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.types.Departure
import com.octara.maps.types.Route
import com.octara.maps.types.RouteType
import com.octara.maps.types.Stop
import com.octara.maps.managers.FavoritesStopsManager
import com.octara.maps.ui.components.RouteIcon
import com.octara.maps.ui.components.RouteTypeIcon
import com.octara.maps.ui.theme.AppDimensions
import com.octara.maps.utils.Api
import com.octara.maps.navigation.Screen
import kotlinx.coroutines.delay

private fun parseDepartureTime(time: String): String {
    val match = Regex("(?:^|T|\\s)(\\d{1,2}):(\\d{2})").find(time)
    return match?.let {
        "${it.groupValues[1].padStart(2, '0')}:${it.groupValues[2]}"
    } ?: ""
}

private fun formatDepartureTime(dep: Departure): String {
    val wait = dep.wait_minutes
    return when {
        wait <= 1.0f -> "Imminent"
        wait < 60.0f -> "${wait.toInt()} min"
        wait < 120.0f -> {
            val hours = (wait / 60).toInt()
            val mins = (wait % 60).toInt()
            if (mins == 0) "${hours} h" else "${hours} h ${mins} min"
        }

        !dep.time_string.isNullOrBlank() -> dep.time_string
        !dep.time.isNullOrBlank() -> parseDepartureTime(dep.time)
        else -> {
            val hours = (wait / 60).toInt()
            val mins = (wait % 60).toInt()
            if (mins == 0) "${hours} h" else "${hours} h ${mins} min"
        }
    }
}

private fun getRouteTypePriority(type: RouteType): Int {
    return when (type) {
        RouteType.TRAIN -> 1
        RouteType.METRO -> 2
        RouteType.TRAM -> 3
        RouteType.BUS -> 4
        else -> 5
    }
}

private fun compareRouteNames(a: String, b: String): Int {
    val aNum = a.filter { it.isDigit() }.toIntOrNull()
    val bNum = b.filter { it.isDigit() }.toIntOrNull()
    if (aNum != null && bNum != null && aNum != bNum) {
        return aNum.compareTo(bNum)
    }
    return a.compareTo(b, ignoreCase = true)
}

data class NextRoutesDeparture(
    val routeId: String,
    val name: String,
    val picto: String?,
    val type: RouteType,
    val departures: List<Departure>,
    val color: String,
    val typeName: String
) {
    fun toRoute(): Route = Route(
        id = routeId,
        short_name = name,
        long_name = name,
        type = type,
        color = color,
        tcolor = "",
        picto = picto
    )
}

@Composable
private fun RouteDepartureCard(
    routeInfo: NextRoutesDeparture,
    stop: Stop? = null,
    stopId: String = stop?.id ?: "",
    favoritesManager: FavoritesStopsManager = FavoritesStopsManager(),
    modifier: Modifier = Modifier
) {
    val currentStopId = stop?.id ?: stopId
    var isFavorite by remember(currentStopId, routeInfo.routeId) {
        mutableStateOf(
            if (stop != null) {
                favoritesManager.isFavorite(stop, routeInfo.toRoute())
            } else {
                favoritesManager.isFavorite(currentStopId, routeInfo.routeId)
            }
        )
    }

    Card(
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f)
        ),
        modifier = modifier.fillMaxWidth()
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                RouteIcon(route = routeInfo.toRoute(), modifier = Modifier.height(28.dp))
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(1.dp)
                ) {
                    Text(
                        text = routeInfo.name,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = routeInfo.typeName,
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                if (routeInfo.departures.isEmpty()) {
                    Text(
                        text = "Aucun départ prévu",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                Icon(
                    imageVector = if (isFavorite) Icons.Default.Star else Icons.Default.StarOutline,
                    contentDescription = if (isFavorite) "Retirer des favoris" else "Mettre en favori",
                    modifier = Modifier
                        .size(20.dp)
                        .clickable {
                            if (stop != null) {
                                favoritesManager.toggleFavoriteStop(
                                    stop = stop,
                                    route = routeInfo.toRoute(),
                                )
                                isFavorite = favoritesManager.isFavorite(stop, routeInfo.toRoute())
                            } else {
                                favoritesManager.toggleFavoriteStop(
                                    stopId = currentStopId,
                                    routeId = routeInfo.routeId,
                                    routeName = routeInfo.name,
                                    routeColor = routeInfo.color,
                                    routepicto = routeInfo.picto
                                )
                                isFavorite = favoritesManager.isFavorite(currentStopId, routeInfo.routeId)
                            }
                        },
                    tint = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            if (routeInfo.departures.isNotEmpty()) {
                HorizontalDivider(
                    color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.35f),
                    modifier = Modifier.padding(vertical = 2.dp)
                )
                for (dep in routeInfo.departures) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 3.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                            contentDescription = null,
                            modifier = Modifier.size(14.dp),
                            tint = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Text(
                            text = if (!dep.headsign.isNullOrBlank()) dep.headsign else "Direction inconnue",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.onSurface,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f)
                        )
                        val waitText = formatDepartureTime(dep)
                        val isImminent = dep.wait_minutes <= 1.0f
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = if (isImminent) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant,
                            modifier = Modifier.padding(start = 4.dp)
                        ) {
                            Text(
                                text = waitText,
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = if (isImminent) FontWeight.Bold else FontWeight.SemiBold,
                                color = if (isImminent) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface,
                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StopInfoSheetContent(
    stopId: String,
    modifier: Modifier = Modifier,
    sheetState: SheetState? = null,
    onClose: (() -> Unit)? = null,
    onStopLoaded: ((lat: Double, lon: Double, name: String) -> Unit)? = null
) {
    val router = LocalRouter.current
    val favoritesManager = com.octara.maps.managers.FavoritesStopsManager()

    if (sheetState != null) {
        LaunchedEffect(sheetState.currentValue) {
            if (sheetState.currentValue == SheetValue.Hidden) {
                onClose?.invoke() ?: router.goBack()
            }
        }
    }

    var isLoading by remember { mutableStateOf(true) }
    var stopInfo by remember { mutableStateOf<Api.StopInfoResponse?>(null) }

    val nextDeparturesLines = remember {
        mutableStateOf<List<NextRoutesDeparture>>(emptyList())
    }
    var refreshTrigger by remember { mutableStateOf(0) }

    LaunchedEffect(stopId) {
        while (true) {
            delay(60_000L)
            refreshTrigger++
        }
    }

    LaunchedEffect(stopId, refreshTrigger) {
        isLoading = true
        val response = com.octara.maps.utils.Api().getStopById(stopId)
        stopInfo = response
        isLoading = false

        if (response != null && response.stop.lat != 0f && response.stop.lon != 0f) {
            onStopLoaded?.invoke(
                response.stop.lat.toDouble(),
                response.stop.lon.toDouble(),
                response.stop.name
            )
        }

        val allDepartures = response?.departures.orEmpty()
        val allRoutes = response?.routes.orEmpty()

        val departuresByRoute = allDepartures.groupBy { dep ->
            dep.route_id?.removePrefix("IDFM:") ?: ""
        }

        val mappedRoutes = allRoutes.map { route ->
            val cleanId = route.id.removePrefix("IDFM:")
            val routeDeps = departuresByRoute[cleanId]
                ?: allDepartures.filter { it.route_name == route.name && it.type == route.type }
            NextRoutesDeparture(
                routeId = route.id,
                name = route.name,
                picto = route.picto,
                type = route.type,
                departures = routeDeps.sortedBy { it.wait_minutes },
                color = route.color,
                typeName = route.typeName
            )
        }

        val knownCleanIds = allRoutes.map { it.id.removePrefix("IDFM:") }.toSet()
        val orphanRoutes = departuresByRoute.filterKeys { it !in knownCleanIds && it.isNotEmpty() }
            .map { (orphanId, orphanDeps) ->
                val firstDep = orphanDeps.first()
                val rType = firstDep.type ?: RouteType.BUS
                val rName = if (!firstDep.route_name.isNullOrBlank()) firstDep.route_name else orphanId
                NextRoutesDeparture(
                    routeId = orphanId,
                    name = rName,
                    picto = firstDep.picto,
                    type = rType,
                    departures = orphanDeps.sortedBy { it.wait_minutes },
                    color = firstDep.color ?: "",
                    typeName = when (rType) {
                        RouteType.TRAM -> "Tramway"
                        RouteType.METRO -> "Métro"
                        RouteType.TRAIN -> "Train"
                        RouteType.BUS -> "Bus"
                        RouteType.FERRY -> "Ferry"
                        RouteType.CABLE_TRAM, RouteType.AERIAL_LIFT -> "Télécabine"
                        RouteType.FUNICULAR -> "Funiculaire"
                    }
                )
            }

        nextDeparturesLines.value = (mappedRoutes + orphanRoutes)
            .sortedWith(
                compareBy<NextRoutesDeparture> {
                    if (it.departures.isNotEmpty()) 0 else 1
                }.thenBy {
                    getRouteTypePriority(it.type)
                }.thenComparator { a, b ->
                    compareRouteNames(a.name, b.name)
                }
            )
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 8.dp)
    ) {
        val info = stopInfo
        if (info == null && isLoading) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 32.dp),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = "Chargement de l'arrêt...",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        } else if (info == null) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 32.dp),
                contentAlignment = Alignment.Center
            ) {
                Text(
                    text = "Erreur lors du chargement de l'arrêt",
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.error
                )
            }
        } else {
            val routes = info.routes ?: emptyList()
            val allDepartures = info.departures ?: emptyList()

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 4.dp, bottom = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Box(
                    modifier = Modifier
                        .size(44.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f)),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.Default.Place,
                        contentDescription = "Arrêt",
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(24.dp)
                    )
                }
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(2.dp)
                ) {
                    Text(
                        text = info.stop.name,
                        style = MaterialTheme.typography.titleLarge,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Text(
                        text = if (routes.isNotEmpty()) {
                            "${routes.size} ligne${if (routes.size > 1) "s" else ""} • ${allDepartures.size} départ${if (allDepartures.size > 1) "s" else ""}"
                        } else {
                            "Arrêt de transport"
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }

            Button(
                onClick = {
                    router.navigateTo(
                        Screen.JourneyPlanner(
                            toStopId = info.stop.id,
                            toStopName = info.stop.name
                        )
                    )
                },
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(12.dp),
                colors = androidx.compose.material3.ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer,
                    contentColor = MaterialTheme.colorScheme.onSurfaceVariant
                )
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "Itineraire",
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier.padding(start = 8.dp)
                    )
                    Icon(
                        imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                        contentDescription = "Itinéraire",
                        modifier = Modifier.size(20.dp)
                    )
                }
            }

            HorizontalDivider(
                color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f),
                modifier = Modifier.padding(bottom = 12.dp)
            )

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 8.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "Prochains départs",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface
                )
            }

            if (nextDeparturesLines.value.isEmpty()) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 24.dp),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = "Aucune ligne disponible",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            } else {
                val (withDepartures, withoutDepartures) = nextDeparturesLines.value.partition { it.departures.isNotEmpty() }

                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    for (nextRoute in withDepartures) {
                        RouteDepartureCard(
                            stop = info.stop,
                            routeInfo = nextRoute,
                            favoritesManager = favoritesManager
                        )
                    }

                    if (withoutDepartures.isNotEmpty()) {
                        if (withDepartures.isNotEmpty()) {
                            Spacer(modifier = Modifier.height(4.dp))
                            HorizontalDivider(
                                color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.4f),
                                modifier = Modifier.padding(vertical = 4.dp)
                            )
                            Text(
                                text = "Autres lignes (aucun départ immédiat)",
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(vertical = 4.dp)
                            )
                        }

                        for (nextRoute in withoutDepartures) {
                            RouteDepartureCard(
                                stop = info.stop,
                                routeInfo = nextRoute,
                                favoritesManager = favoritesManager
                            )
                        }
                    }
                }
            }
        }
    }
}

