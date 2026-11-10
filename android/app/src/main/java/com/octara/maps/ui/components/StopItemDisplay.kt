package com.octara.maps.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shadow
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.octara.maps.types.LayerStop
import com.octara.maps.types.LayerStopRoute
import com.octara.maps.types.Route
import com.octara.maps.types.RouteType
import com.octara.maps.types.StopType
import com.octara.maps.ui.components.RouteIcon
import com.octara.maps.ui.components.getStopIconResource

private fun inferFallbackRouteTypeAndColor(
    routeName: String,
    explicitType: RouteType?,
    explicitColor: String,
    stopType: StopType
): Pair<RouteType, String> {
    val cleanName = routeName.trim().uppercase()
    val inferredType = explicitType ?: when {
        cleanName in setOf("A", "B", "C", "D", "E", "H", "J", "K", "L", "N", "P", "R", "U", "V") -> RouteType.TRAIN
        cleanName in setOf(
            "1",
            "2",
            "3",
            "3B",
            "3BIS",
            "4",
            "5",
            "6",
            "7",
            "7B",
            "7BIS",
            "8",
            "9",
            "10",
            "11",
            "12",
            "13",
            "14"
        ) -> RouteType.METRO

        cleanName.startsWith("T") && cleanName.drop(1).any { it.isDigit() } -> RouteType.TRAM
        stopType == StopType.RAIL -> RouteType.TRAIN
        stopType == StopType.METRO -> RouteType.METRO
        stopType == StopType.STREET_TRAM -> RouteType.TRAM
        else -> RouteType.BUS
    }

    val inferredColor = if (explicitColor.isNotBlank()) {
        explicitColor
    } else {
        when (cleanName) {
            "A" -> "E3051C"
            "B" -> "5291CE"
            "C" -> "FFCE00"
            "D" -> "00814F"
            "E" -> "C04191"
            "H" -> "8D5E2A"
            "J" -> "D5C900"
            "K" -> "9F9825"
            "L" -> "CEADD2"
            "N" -> "00A88F"
            "P" -> "F28E42"
            "R" -> "E59FDB"
            "U" -> "B90845"
            "V" -> "9F9825"
            "1" -> "FFCE00"
            "2" -> "0064B0"
            "3" -> "9F9825"
            "3B", "3BIS" -> "98D4E2"
            "4" -> "C04191"
            "5" -> "F28E42"
            "6" -> "83C491"
            "7" -> "F3A4BA"
            "7B", "7BIS" -> "83C491"
            "8" -> "CEADD2"
            "9" -> "D5C900"
            "10" -> "E3B32A"
            "11" -> "8D5E2A"
            "12" -> "00814F"
            "13" -> "98D4E2"
            "14" -> "662483"
            else -> "0064B0"
        }
    }

    return Pair(inferredType, inferredColor)
}

@Composable
fun StopPictoItem(
    route: LayerStopRoute,
    stopType: StopType = StopType.UNKNOWN,
    modifier: Modifier = Modifier.height(18.dp)
) {
    val (rType, rColor) = inferFallbackRouteTypeAndColor(
        routeName = route.name,
        explicitType = route.type,
        explicitColor = route.color,
        stopType = stopType
    )

    RouteIcon(
        route = Route(
            id = route.name,
            short_name = route.name,
            type = rType,
            color = rColor,
            tcolor = "",
            picto = route.picto
        ),
        modifier = modifier,
        height = 18.dp
    )
}

@Composable
fun StopItemDisplay(
    stop: LayerStop,
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    withBG: Boolean = false
) {
    Row(
        modifier = modifier
            .then(if (onClick != null) Modifier.clickable { onClick() } else Modifier),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp)
    ) {
        Icon(
            imageVector = getStopIconResource(stop.type),
            contentDescription = "Stop type",
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier
                .size(20.dp)
                .then(
                    if (withBG) {
                        Modifier
                            .clip(RoundedCornerShape(4.dp))
                            .background(Color.White)
                            .padding(2.dp)
                    } else {
                        Modifier
                    }
                )
        )

        Column(
            verticalArrangement = Arrangement.spacedBy(1.dp)
        ) {
            Text(
                text = stop.name,
                style = MaterialTheme.typography.labelMedium.copy(
                    shadow = Shadow(
                        color = Color.White,
                        offset = Offset(0f, 0f),
                        blurRadius = 6f
                    )
                ),
                fontSize = 11.sp,
                fontWeight = FontWeight.Bold,
                color = Color(0xFF1A1A1A),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            val routes = stop.routes
            if (routes.isNotEmpty()) {
                val maxVisible = 4
                val hasOverflow = routes.size > maxVisible
                val displayedRoutes = if (hasOverflow) routes.take(3) else routes.take(maxVisible)
                val overflowCount = routes.size - displayedRoutes.size

                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(3.dp)
                ) {
                    displayedRoutes.forEach { r ->
                        StopPictoItem(
                            route = r,
                            stopType = stop.type,
                            modifier = Modifier.height(18.dp)
                        )
                    }

                    if (hasOverflow && overflowCount > 0) {
                        Box(
                            modifier = Modifier
                                .height(18.dp)
                                .clip(RoundedCornerShape(4.dp))
                                .background(MaterialTheme.colorScheme.surfaceVariant)
                                .padding(horizontal = 4.dp),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                text = "+$overflowCount",
                                style = MaterialTheme.typography.labelSmall.copy(
                                    platformStyle = androidx.compose.ui.text.PlatformTextStyle(
                                        includeFontPadding = false
                                    ),
                                    lineHeight = 10.sp
                                ),
                                fontSize = 10.sp,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    }
                }
            }
        }
    }
}
